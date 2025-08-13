package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"imageProcessor/internal/config"
	"imageProcessor/internal/http-server/handlers/image/deleteImage"
	"imageProcessor/internal/http-server/handlers/image/getImage"
	"imageProcessor/internal/http-server/handlers/image/saveImage"
	"imageProcessor/internal/http-server/middleware/mwlogger"
	"imageProcessor/internal/kafka/consumer"
	"imageProcessor/internal/kafka/producer"
	"imageProcessor/internal/lib/logger/handlers/slogpretty"
	"imageProcessor/internal/lib/logger/sl"
	"imageProcessor/internal/processor"
	"imageProcessor/internal/storage/postgres"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("Starting image processor", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	storage, err := postgres.InitDB(&cfg.Database)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	kafkaProducer, err := producer.NewProducer(&cfg.Kafka, log)
	if err != nil {
		log.Error("failed to create kafka producer", sl.Err(err))
		os.Exit(1)
	}

	kafkaConsumer, err := consumer.NewConsumer(&cfg.Kafka, log)
	if err != nil {
		log.Error("failed to create kafka consumer", sl.Err(err))
		os.Exit(1)
	}

	imageProcessor := processor.NewImageProcessor(log, storage)

	go kafkaConsumer.ReadMessages(context.Background(), imageProcessor.ProcessMessage)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwlogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Handle("/", http.FileServer(http.Dir("./static")))

	router.Handle("/processed/*", http.StripPrefix("/processed/", http.FileServer(http.Dir("./processed"))))

	router.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	router.Post("/upload", saveImage.New(log, storage, kafkaProducer))
	router.Get("/image/{id}", getImage.New(log, storage))
	router.Delete("/image/{id}", deleteImage.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err = srv.ListenAndServe(); err != nil {
		log.Error("failed to start server", sl.Err(err))
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	sign := <-stop

	log.Info("application stopping", slog.String("signal", sign.String()))

	log.Info("application stopped")

	if err = storage.Close(); err != nil {
		log.Error("failed to close database", slog.String("error", err.Error()))
	}

	log.Info("postgres connection closed")

	if err = kafkaProducer.Close(); err != nil {
		log.Error("failed to close kafka producer", slog.String("error", err.Error()))
	}

	if err = kafkaConsumer.Close(); err != nil {
		log.Error("failed to close kafka consumer", slog.String("error", err.Error()))
	}

	log.Info("kafka connection closed")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	h := opts.NewPrettyHandler(os.Stdout)

	return slog.New(h)
}

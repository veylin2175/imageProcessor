FROM golang:1.23.6-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /image-processor ./cmd/image-processor

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /image-processor /app/image-processor

COPY ./config ./config
COPY ./static ./static
COPY ./watermark.png ./watermark.png

RUN mkdir -p uploads processed

RUN chmod +x /app/image-processor

EXPOSE 8075

CMD ["/app/image-processor"]
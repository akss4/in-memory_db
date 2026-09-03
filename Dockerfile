FROM golang:1.27 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o in-memory-db .

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/in-memory-db .

EXPOSE 6379

CMD ["./in-memory-db"]
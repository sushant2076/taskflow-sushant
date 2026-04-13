# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server /server
COPY --from=builder /app/migrations /migrations
COPY --from=builder /app/seed /seed

EXPOSE 8080

ENTRYPOINT ["/server"]

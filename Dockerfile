# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod first for better layer caching (no external deps, so no go.sum needed)
COPY go.mod ./

COPY *.go ./
COPY static ./static

RUN CGO_ENABLED=0 GOOS=linux go build -o ticket-system .

# ---- Final stage ----
FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/ticket-system .

EXPOSE 8080

CMD ["./ticket-system"]
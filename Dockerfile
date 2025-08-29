# --- Build stage ---
FROM golang:1.22.5 as builder
WORKDIR /app
COPY . .
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o openapimcp ./cmd/openapi-mcp
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/openapimcp  main.go

# --- Runtime stage ---
FROM alpine:latest
RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/bin/openapimcp ./openapimcp
# Copy the specs directory instead of individual files
COPY ./specs ./specs
RUN chmod +x ./openapimcp
EXPOSE 8080
CMD ["/app/openapimcp"]

# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
# We use -ldflags="-s -w" to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o boligaksjonen .

# Stage 2: Create a minimal production image
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/boligaksjonen .

# Copy static assets and views
COPY --from=builder /app/static ./static
COPY --from=builder /app/views ./views

# Expose the application port
EXPOSE 8080

# Run the application
CMD ["./boligaksjonen"]

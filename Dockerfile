# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy everything including vendor directory
COPY . .

# Build the application using vendor mode
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o poppit .

# Runtime stage
FROM alpine:latest

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/poppit .

# Run the application
CMD ["./poppit"]

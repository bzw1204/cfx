# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Build dependencies
RUN apk add --no-cache git ca-certificates

# Cache module downloads
COPY go.mod go.sum ./
RUN go mod download

# Build the binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o cfx .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/cfx /usr/local/bin/cfx

ENTRYPOINT ["cfx"]

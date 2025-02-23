FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o mtr-tool

FROM alpine:latest

# Install mtr
RUN apk add --no-cache mtr

WORKDIR /app
COPY --from=builder /app/mtr-tool .

EXPOSE 8080
CMD ["./mtr-tool"]

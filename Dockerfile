FROM golang:1.17.8 AS builder
WORKDIR /app
COPY main.go go.mod ./
# Disable CGO to make a minimal image with golang
RUN CGO_ENABLED=0 GOOS=linux go build -o lb .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root
COPY --from=builder /app/lb .
ENTRYPOINT ["/root/lb"]
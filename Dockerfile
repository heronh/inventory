FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o inventory ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates docker-cli docker-cli-compose
COPY --from=builder /app/inventory /app/inventory
COPY --from=builder /app/templates /app/templates
COPY --from=builder /app/.env /app/.env
COPY --from=builder /app/docker-compose.yml /app/docker-compose.yml
COPY --from=builder /app/docker /app/docker

EXPOSE 8008
CMD ["/app/inventory"]
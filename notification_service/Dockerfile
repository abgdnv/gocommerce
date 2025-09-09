FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY notification_service/go.mod notification_service/go.sum ./notification_service/
COPY pkg/ ./pkg/

WORKDIR /app/notification_service
RUN go mod download

WORKDIR /app
COPY notification_service/ ./notification_service/

WORKDIR /app/notification_service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app github.com/abgdnv/gocommerce/notification_service/cmd/

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/notification_service/app .

RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
USER appuser

EXPOSE 8080

CMD ["./app"]

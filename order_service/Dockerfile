FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY order_service/go.mod order_service/go.sum ./order_service/
COPY pkg/ ./pkg/

WORKDIR /app/order_service
RUN go mod download

WORKDIR /app
COPY order_service/ ./order_service/

WORKDIR /app/order_service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app github.com/abgdnv/gocommerce/order_service/cmd/

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/order_service/app .

RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
USER appuser

EXPOSE 8080

CMD ["./app"]

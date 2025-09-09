FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY product_service/go.mod product_service/go.sum ./product_service/
COPY pkg/ ./pkg/

WORKDIR /app/product_service
RUN go mod download

WORKDIR /app
COPY product_service/ ./product_service/

WORKDIR /app/product_service
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app github.com/abgdnv/gocommerce/product_service/cmd/

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/product_service/app .

RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
USER appuser

EXPOSE 8080

CMD ["./app"]

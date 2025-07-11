FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY order_service/go.mod order_service/go.sum ./
RUN go mod download

COPY pkg/ ./pkg/
COPY order_service/ ./

RUN go mod edit -replace "github.com/abgdnv/gocommerce/pkg=./pkg"
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app github.com/abgdnv/gocommerce/order_service/cmd/

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/app .

RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
USER appuser

EXPOSE 8080

CMD ["./app"]

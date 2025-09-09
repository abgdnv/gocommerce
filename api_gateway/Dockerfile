FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY api_gateway/go.mod api_gateway/go.sum ./api_gateway/
COPY pkg/ ./pkg/

WORKDIR /app/api_gateway
RUN go mod download

WORKDIR /app
COPY api_gateway/ ./api_gateway/

WORKDIR /app/api_gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app github.com/abgdnv/gocommerce/api_gateway/cmd/

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/api_gateway/app .

RUN adduser -D -g '' appuser && chown appuser:appuser /app/app
USER appuser

EXPOSE 8080

CMD ["./app"]

.PHONY: proto.gen

# Generate Go code from Protobuf definitions
proto.gen:
	@protoc \
		--proto_path=proto \
		--go_out=gen/go --go_opt=paths=source_relative \
		--go-grpc_out=gen/go --go-grpc_opt=paths=source_relative \
		proto/product/v1/product.proto

	@echo "✅ Protobuf code generated"

.PHONY: sqlc.gen

sqlc.gen:
	@sqlc generate -f internal/product/store/sqlc.yaml

	@echo "✅ sqlc code generated"
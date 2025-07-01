.PHONY: proto.gen

# Generate Go code from Protobuf definitions
proto.gen:
	@protoc \
		--proto_path=proto \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/product/v1/product.proto

	@echo "✅ Protobuf code generated"
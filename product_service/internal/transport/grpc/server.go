// Package grpc provides a gRPC server for the product service.
package grpc

import (
	"context"
	"log/slog"

	pb "github.com/abgdnv/gocommerce/pkg/api/gen/go/product/v1"
	"github.com/abgdnv/gocommerce/product_service/internal/service"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProductService defines the interface for the product service.
type ProductService interface {
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]service.ProductDto, error)
}

type Server struct {
	// Embed the unimplemented server for forward compatibility
	pb.UnimplementedProductServiceServer
	service ProductService
}

func NewServer(service ProductService) *Server {
	return &Server{service: service}
}

func (s *Server) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	logger := slog.With(slog.Any("product_ids", req.Products))
	logger.Info("received grpc request GetProduct")
	ids := make([]uuid.UUID, 0, len(req.Products))
	for _, item := range req.Products {
		id, err := uuid.Parse(item)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
		}
		ids = append(ids, id)
	}

	found, err := s.service.FindByIDs(ctx, ids)
	if err != nil {
		logger.Error("service.FindByIDs failed", slog.Any("error", err))
		return nil, status.Errorf(codes.Internal, "internal server error")
	}
	if len(found) < len(ids) {
		return nil, status.Errorf(codes.NotFound, "at least one of the products is not found")
	}

	products := make([]*pb.Product, 0, len(req.Products))
	for _, product := range found {
		products = append(products, &pb.Product{
			Id:            product.ID,
			Name:          product.Name,
			Price:         product.Price,
			StockQuantity: product.Stock,
			Version:       product.Version,
		})
	}
	logger.Info("send grpc response for GetProduct")
	return &pb.GetProductResponse{
		Products: products,
	}, nil
}

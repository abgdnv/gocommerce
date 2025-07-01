// Package grpc provides a gRPC server for the product service.
package grpc

import (
	"context"
	"errors"
	"log/slog"

	perrors "github.com/abgdnv/gocommerce/internal/product/errors"
	"github.com/abgdnv/gocommerce/internal/product/service"
	pb "github.com/abgdnv/gocommerce/product/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProductService defines the interface for the product service.
type ProductService interface {
	FindByID(ctx context.Context, id uuid.UUID) (*service.ProductDto, error)
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
	id, err := uuid.Parse(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid product ID: %v", err)
	}
	logger := slog.With(slog.String("product_id", id.String()))
	product, err := s.service.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, perrors.ErrProductNotFound) {
			return nil, status.Errorf(codes.NotFound, "product with id %s not found", id.String())
		}
		logger.Error("service.FindByID failed", slog.Any("error", err))
		return nil, status.Errorf(codes.Internal, "internal server error")
	}

	return &pb.GetProductResponse{
		Product: &pb.Product{
			Id:            product.ID,
			Name:          product.Name,
			Price:         product.Price,
			StockQuantity: product.Stock,
			Version:       product.Version,
		},
	}, nil
}

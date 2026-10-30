package rpc

import (
	"context"
	"time"

	pb "github.com/nilesh/shopcore-contracts/shopcorepb/shopcorepb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Clients struct {
	User        pb.UserServiceClient
	Product     pb.ProductServiceClient
	userConn    *grpc.ClientConn
	productConn *grpc.ClientConn
}

func NewClients(userAddress, productAddress string) (*Clients, error) {
	userConn, err := grpc.NewClient(userAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	productConn, err := grpc.NewClient(productAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		userConn.Close()
		return nil, err
	}
	return &Clients{
		User:        pb.NewUserServiceClient(userConn),
		Product:     pb.NewProductServiceClient(productConn),
		userConn:    userConn,
		productConn: productConn,
	}, nil
}

func (c *Clients) UserDetails(ctx context.Context, userID string) (*pb.UserResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.User.GetUserDetails(ctx, &pb.UserRequest{UserId: userID})
}

func (c *Clients) ProductDetails(ctx context.Context, productID string) (*pb.ProductResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return c.Product.GetProductDetails(ctx, &pb.ProductRequest{ProductId: productID})
}

func (c *Clients) Close() {
	_ = c.userConn.Close()
	_ = c.productConn.Close()
}

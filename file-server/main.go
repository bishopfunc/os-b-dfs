package main

import (
	"context"
	"log"
	"net"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedFileServiceServer
}

func (s *server) Read(ctx context.Context, in *pb.FileRequest) (*pb.FileResponse, error) {
	return &pb.FileResponse{Content: "Dummy file content"}, nil
}

func (s *server) Write(ctx context.Context, in *pb.FileRequest) (*pb.FileResponse, error) {
	return &pb.FileResponse{Content: "Write success"}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFileServiceServer(s, &server{})
	log.Println("Server listening on port 50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

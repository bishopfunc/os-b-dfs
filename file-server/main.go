package main

import (
	"context"
	"io/ioutil"
	"log"
	"net"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
)

var (
	cacheDir = map[string][]string{
		"a,txt": []string{"localhost:50052", "localhost:50053"},
	}
)

type server struct {
	pb.UnimplementedDFSServer
}

func (s *server) Read(ctx context.Context, in *pb.FileRequest) (*pb.FileResponse, error) {
	return &pb.FileResponse{Content: "Dummy file content"}, nil
}

func (s *server) Write(ctx context.Context, in *pb.FileRequest) (*pb.FileResponse, error) {
	return &pb.FileResponse{Content: "Write success"}, nil
}

func (s *server) OpenFile(ctx context.Context, in *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	content, err := ioutil.ReadFile(in.Filename)
	if err != nil {
		return nil, err
	}
	return &pb.OpenFileResponse{Content: string(content)}, nil
}

// 各clientが持ってるcacheを更新する
func (s *server) UpdateCache(ctx context.Context, in *pb.UpdateCacheRequest) (*pb.UpdateCacheResponse, error) {
	cacheDir[in.Filename] = append(cacheDir[in.Filename], in.Client)
	return &pb.UpdateCacheResponse{Success: true}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterDFSServer(s, &server{})
	log.Println("Server listening on port 50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

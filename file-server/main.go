package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
)

var (
	cacheDir = map[string][]string{}
	lockDir  = map[string]bool{}
)

// {"a.txt": ["localhost:50052", "localhost:50053"], "b.txt": ["localhost:50052"]}

type server struct {
	pb.UnimplementedDFSServer
}

func (s *server) ReadFile(ctx context.Context, rfr *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	content, err := os.ReadFile(rfr.Filename)
	if err != nil {
		return nil, fmt.Errorf("[server] failed to read file: %v", err)
	}
	return &pb.ReadFileResponse{Content: string(content)}, nil
}

func (s *server) WriteFile(ctx context.Context, wfr *pb.WriteFileRequest) (*pb.WriteFileResponse, error) {
	err := os.WriteFile(wfr.Filename, []byte(wfr.Content), 0644)
	return &pb.WriteFileResponse{Success: err == nil}, nil
}

func (s *server) OpenFile(ctx context.Context, ofr *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	content, err := ioutil.ReadFile(ofr.Filename)
	if err != nil {
		return nil, fmt.Errorf("[server] failed to open file: %v", err)
	}
	return &pb.OpenFileResponse{Content: string(content)}, nil
}

// 各clientが持ってるcacheを更新する
func (s *server) UpdateCache(ctx context.Context, ucr *pb.UpdateCacheRequest) (*pb.UpdateCacheResponse, error) {
	cacheDir[ucr.Filename] = append(cacheDir[ucr.Filename], ucr.Client)
	return &pb.UpdateCacheResponse{Success: true}, nil
}

func (s *server) DeleteCache(ctx context.Context, dcr *pb.DeleteCacheRequest) (*pb.DeleteCacheResponse, error) {
	delete(cacheDir, dcr.Filename) // cacheDirから削除する
	return &pb.DeleteCacheResponse{Success: true}, nil
}

func (s *server) UpdateLock(ctx context.Context, ulr *pb.UpdateLockRequest) (*pb.UpdateLockResponse, error) {
	lockDir[ulr.Filename] = ulr.Lock
	return &pb.UpdateLockResponse{Success: true}, nil
}

func (s *server) CheckcLock(ctx context.Context, clr *pb.CheckLockRequest) (*pb.CheckLockResponse, error) {
	return &pb.CheckLockResponse{Locked: lockDir[clr.Filename]}, nil
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

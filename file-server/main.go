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

func (s *server) ReadFile(ctx context.Context, in *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	content, _ := os.ReadFile(in.Filename)
	return &pb.ReadFileResponse{Content: string(content)}, nil
}

func (s *server) WriteFile(ctx context.Context, in *pb.WriteFileRequest) (*pb.WriteFileResponse, error) {
	err := os.WriteFile(in.Filename, []byte(in.Content), 0644)
	return &pb.WriteFileResponse{Success: err == nil}, nil
}

func (s *server) OpenFile(ctx context.Context, in *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	content, err := ioutil.ReadFile(in.Filename)
	if err != nil {
		return nil, fmt.Errorf("[server] failed to open file: %v", err)
	}
	return &pb.OpenFileResponse{Content: string(content)}, nil
}

// 各clientが持ってるcacheを更新する
func (s *server) UpdateCache(ctx context.Context, in *pb.UpdateCacheRequest) (*pb.UpdateCacheResponse, error) {
	cacheDir[in.Filename] = append(cacheDir[in.Filename], in.Client)
	return &pb.UpdateCacheResponse{Success: true}, nil
}

func (s *server) DeleteCache(ctx context.Context, in *pb.DeleteCacheRequest) (*pb.DeleteCacheResponse, error) {
	delete(cacheDir, in.Filename) // cacheDirから削除する
	return &pb.DeleteCacheResponse{Success: true}, nil
}

func (s *server) UpdateLock(ctx context.Context, in *pb.UpdateLockRequest) (*pb.UpdateLockResponse, error) {
	lockDir[in.Filename] = in.Lock
	return &pb.UpdateLockResponse{Success: true}, nil
}

func (s *server) CheckLock(ctx context.Context, in *pb.CheckLockRequest) (*pb.CheckLockResponse, error) {
	return &pb.CheckLockResponse{Locked: lockDir[in.Filename]}, nil
}

func (s *server) InvalidNotification(req *pb.InvalidNotificationRequest, stream pb.DFS_InvalidNotificationServer) error {
	clientList := cacheDir[req.Filename]
	for _, client := range clientList {
		if req.Except != nil && req.Except.Value == client {
			fmt.Printf("except: %s\n", req.Except.Value)
			continue
		}
		fmt.Printf("client: %s\n", client)
		err := stream.Send(&pb.InvalidNotificationResponse{Invalid: true})
		if err != nil {
			return fmt.Errorf("[server] failed to send invalid notification: %v", err)
		}
	}
	return nil
}

func startServer(port string) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", port, err)
	}

	s := grpc.NewServer()
	pb.RegisterDFSServer(s, &server{})

	log.Printf("Server listening on %s", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve on %s: %v", port, err)
	}
}

func main() {
	go startServer(":50052")
	go startServer(":50053")
	select {} // メインゴルーチンをブロックし続ける
}

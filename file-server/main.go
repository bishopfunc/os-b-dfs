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
	// clients: key:uuid, value:pb.DFS_InvalidNotificationServer
	clients = map[string]pb.DFS_InvalidNotificationServer{}
	// cacheDict: key:fileName, value:uuid
	cacheDict = map[string][]string{}
	// dirty or clean
	statusDict map[string]bool
	lockDir  = map[string]bool{}
)

// {"a.txt": ["localhost:50052", "localhost:50053"], "b.txt": ["localhost:50052"]}

type server struct {
	pb.UnimplementedDFSServer
}

func (s *server) ReadFile(ctx context.Context, in *pb.ReadFileRequest) (*pb.ReadFileResponse, error) {
	content, err := os.ReadFile(in.Filename)
	if err != nil {
		return nil, fmt.Errorf("[server] failed to read file: %v", err)
	}
	return &pb.ReadFileResponse{Content: string(content)}, nil
}

func (s *server) WriteFile(ctx context.Context, in *pb.WriteFileRequest) (*pb.WriteFileResponse, error) {
	if err := os.WriteFile(in.Filename, []byte(in.Content), 0644); err != nil {
		return nil, fmt.Errorf("[server] failed to write file: %v", err)
	}
	return &pb.WriteFileResponse{Success: true}, nil
}

func (s *server) OpenFile(ctx context.Context, in *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	content, err := ioutil.ReadFile(in.Filename)
	if err != nil {
		return nil, fmt.Errorf("[server] failed to open file: %v", err)
	}
	return &pb.OpenFileResponse{Content: string(content)}, nil
}

func (s *server) CloseFile(ctx context.Context, in *pb.CloseFileRequest) (*pb.CloseFileResponse, error) {
	ulr, err:= s.UpdateLock(ctx, &pb.UpdateLockRequest{Filename: in.Filename, Lock: false})
	if err != nil {
		return nil, fmt.Errorf("[server] failed to close file: %v", err)
	}
	return &pb.CloseFileResponse{Success: ulr.Success}, nil
}

// 各clientが持ってるcacheを更新する
func (s *server) UpdateCache(ctx context.Context, in *pb.UpdateCacheRequest) (*pb.UpdateCacheResponse, error) {
	cacheDict[in.Filename] = append(cacheDict[in.Filename], in.Uuid)
	return &pb.UpdateCacheResponse{Success: true}, nil
}

func (s *server) DeleteCache(ctx context.Context, in *pb.DeleteCacheRequest) (*pb.DeleteCacheResponse, error) {
	delete(cacheDict, in.Filename) // cacheDictから削除する
	return &pb.DeleteCacheResponse{Success: true}, nil
}

func (s *server) UpdateLock(ctx context.Context, in *pb.UpdateLockRequest) (*pb.UpdateLockResponse, error) {
	lockDir[in.Filename] = in.Lock
	return &pb.UpdateLockResponse{Success: true}, nil
}

func (s *server) CheckLock(ctx context.Context, in *pb.CheckLockRequest) (*pb.CheckLockResponse, error) {
	return &pb.CheckLockResponse{Locked: lockDir[in.Filename]}, nil
}

func (s *server) InvalidNotification(srv pb.DFS_InvalidNotificationServer) error {
	for {
		resp, err := srv.Recv()
		if err != nil {
			log.Printf("recv err: %v", err)
			break
		}
		clientUuidList := cacheDict[resp.Filename]
		for _, clientUuid := range clientUuidList {
			client := clients[clientUuid]
			// if resp.Except != nil && resp.Except.Value == client {
			// 	fmt.Printf("except: %s\n", resp.Except.Value)
			// 	continue
			// }
			fmt.Printf("client: %s\n", client)
			if err := client.Send(&pb.InvalidNotificationResponse{Invalid: true}); err != nil {
				return fmt.Errorf("[server] failed to send invalid notification: %v", err)
			}
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

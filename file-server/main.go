package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
)

var (
	// clientServersMap: key:uuid, value:pb.DFS_InvalidNotificationServer
	clientServersMap = make(map[string]pb.DFS_InvalidNotificationServer)
	// haveCacheUserIDsMap: key:fileName, value:[]{}uuid
	haveCacheUserIDsMap = make(map[string][]string)
	// dirty or clean
	// statusMap map[string]bool
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
	haveCacheUserIDsMap[in.Filename] = append(haveCacheUserIDsMap[in.Filename], in.Uuid)
	return &pb.UpdateCacheResponse{Success: true}, nil
}

func (s *server) DeleteCache(ctx context.Context, in *pb.DeleteCacheRequest) (*pb.DeleteCacheResponse, error) {
	delete(haveCacheUserIDsMap, in.Filename) // haveCacheUsersMapから削除する
	return &pb.DeleteCacheResponse{Success: true}, nil
}

func (s *server) UpdateLock(ctx context.Context, in *pb.UpdateLockRequest) (*pb.UpdateLockResponse, error) {
	lockDir[in.Filename] = in.Lock
	return &pb.UpdateLockResponse{Success: true}, nil
}

func (s *server) CheckLock(ctx context.Context, in *pb.CheckLockRequest) (*pb.CheckLockResponse, error) {
	return &pb.CheckLockResponse{Locked: lockDir[in.Filename]}, nil
}

func (s *server) addClient(uid string, srv pb.DFS_InvalidNotificationServer) {
	// s.mu.Lock()
	// defer s.mu.Unlock()
	clientServersMap[uid] = srv
}

func (s *server) removeClient(uid string) {
	// s.mu.Lock()
	// defer s.mu.Unlock()
	delete(clientServersMap, uid)
}

func (s *server) InvalidNotification(srv pb.DFS_InvalidNotificationServer) error {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic: %v", err)
			os.Exit(1)
		}
	}()

	for {
		log.Println("invalid notification called")
		res, err := srv.Recv()
		if err != nil {
			log.Printf("recv err: %v", err)
			break
		}
		// 接続クライアントリストに登録
		s.addClient(res.GetUid(), srv)
		// 関数を抜けるときはリストから削除
		defer s.removeClient(res.GetUid())

		clientUuidList := haveCacheUserIDsMap[res.Filename]
		for _, clientUuid := range clientUuidList {
			if clientUuid == res.GetUid() {
				continue
			}
			client := clientServersMap[clientUuid]
			if client == nil {
				continue
			}
			if err := client.Send(&pb.InvalidNotificationResponse{Filename: res.GetFilename()}); err != nil {
				return fmt.Errorf("[server] failed to send invalid notification: %v", err)
			}
			log.Print("sent invalid notification")
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
	// go startServer(":50053")
	// 10秒ごとにlogを出力
	// デバッグ用
	go func() {
		for {
			log.Printf("clientServersMap: %s\n", clientServersMap)
			log.Printf("haveCacheUserIDsMap: %s\n", haveCacheUserIDsMap)
			log.Println(haveCacheUserIDsMap["a.txt"])
			log.Println("========================================")
			<-time.After(10 * time.Second)
		}
	}()
	select {} // メインゴルーチンをブロックし続ける
}

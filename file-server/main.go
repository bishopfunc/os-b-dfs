package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
)

var (
	// clientServersMap: key:uuid, value:pb.DFS_NotifyInvalidServer
	clientServersMap = make(map[string]pb.DFS_NotifyInvalidServer)
	// haveCacheUserIDsMap: key:fileName, value:[]{}uuid
	haveCacheUserIDsMap = make(map[string][]string)
	lockDir             = map[string]bool{}
)

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

func (s *server) CreateDir(ctx context.Context, in *pb.CreateDirRequest) (*pb.CreateDirResponse, error) {
	err := os.MkdirAll(in.Filepath, 0755)
	if err != nil {
		fmt.Println("[server] failed to make directory:", err)
		return nil, err
	}
	return &pb.CreateDirResponse{Success: true}, nil
}

func (s *server) OpenFile(ctx context.Context, in *pb.OpenFileRequest) (*pb.OpenFileResponse, error) {
	content, err := os.ReadFile(in.Filename)
	if err != nil {
		return nil, fmt.Errorf("[server] failed to open file: %v", err)
	}
	return &pb.OpenFileResponse{Content: string(content)}, nil
}

func (s *server) CloseFile(ctx context.Context, in *pb.CloseFileRequest) (*pb.CloseFileResponse, error) {
	ulr, err := s.UpdateLock(ctx, &pb.UpdateLockRequest{Filename: in.Filename, Lock: false})
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

func (s *server) FilePathWalk(ctx context.Context, in *pb.FilePathWalkRequest) (*pb.FilePathWalkResponse, error) {
	var files []string
	err := filepath.Walk(in.Filepath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if ext == ".go" || ext == ".yaml" {
			return nil // Skip .go and .yaml files
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("[server] failed to walk filepath: %v", err)
	}
	return &pb.FilePathWalkResponse{Filenames: files}, nil
}

func (s *server) NotifyInvalid(srv pb.DFS_NotifyInvalidServer) error {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic: %v", err)
			os.Exit(1)
		}
	}()

	for {
		res, err := srv.Recv()
		if err != nil {
			log.Printf("recv err: %v", err)
			break
		}
		// 接続クライアントリストに登録
		s.addClient(res.GetUid(), srv)
		// 関数を抜けるときはリストから削除
		defer s.removeClient(res.GetUid())

		// 最初1回だけFilenameが空の偽リクエストが送られてくる。この時、接続クライアントリストに登録するだけで良いためここでcontinueする
		if res.GetFilename() == "" {
			continue
		}

		clientUuidList := haveCacheUserIDsMap[res.GetFilename()]
		for _, clientUuid := range clientUuidList {
			if clientUuid == res.GetUid() {
				continue
			}
			client := clientServersMap[clientUuid]
			if client == nil {
				continue
			}
			if err := client.Send(&pb.NotifyInvalidResponse{Filename: res.GetFilename()}); err != nil {
				return fmt.Errorf("[server] failed to send invalid notification: %v", err)
			}
		}
	}

	return nil
}

func (s *server) addClient(uid string, srv pb.DFS_NotifyInvalidServer) {
	clientServersMap[uid] = srv
}

func (s *server) removeClient(uid string) {
	delete(clientServersMap, uid)
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
			log.Println("========================================")
			<-time.After(10 * time.Second)
		}
	}()
	select {} // メインゴルーチンをブロックし続ける
}

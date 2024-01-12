package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	port       = 50052
	host       = "localhost"
	clientName = fmt.Sprintf("%s:%d", host, port)
)

// grpcClient interface
type ClientWrapper struct {
	client pb.DFSClient
	ctx    context.Context
}

func NewClientWrapper(client pb.DFSClient, ctx context.Context) *ClientWrapper {
	return &ClientWrapper{
		client: client,
		ctx:    ctx,
	}
}

func (w *ClientWrapper) OpenAsReadWithoutCache(filename string) (*os.File, error) {
	fileResponse, err := w.client.OpenFile(w.ctx, &pb.OpenFileRequest{Filename: filename}) // w.clinet.Hoge()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	_, _ = w.client.UpdateCache(w.ctx, &pb.UpdateCacheRequest{Filename: filename, Client: clientName})
	// create file
	file, _ := os.Create(filename)
	// write file
	err = os.WriteFile(filename, []byte(fileResponse.GetContent()), 0644)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	fmt.Printf("create cache: %s\n", filename)
	return file, err
}

func (w *ClientWrapper) OpenAsReadWithCache(filename string) (*os.File, error) {
	file, err := os.Open(filename)
	return file, err
}

func (w *ClientWrapper) OpenAsWriteWithoutCache(filename string) (*os.File, error) {
	// lock file
	_, _ = w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true})
	// send invalid from server to other client, other clinet delete cache
	// TODO
	// w.client.SendInvalid(w.ctx, &pb.SendInvalidRequest{Filename: filename, except: clientName})
	// delete cache
	_, _ = w.client.DeleteCache(w.ctx, &pb.DeleteCacheRequest{Filename: filename})
	// create file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return file, err
}

func (w *ClientWrapper) OpenAsWriteWithCache(filename string) (*os.File, error) {
	// lock file
	_, _ = w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true})
	// send invalid from server to other client, other clinet delete cache
	// TODO
	// w.client.SendInvalid(w.ctx, &pb.SendInvalidRequest{Filename: filename, except: clientName})
	// delete cache
	_, _ = w.client.DeleteCache(w.ctx, &pb.DeleteCacheRequest{Filename: filename})
	// create file
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return file, err
}

func (w *ClientWrapper) FinalizeWrite(file *os.File) error {
	filename := file.Name()
	// send file content to server
	fileContent, _ := os.ReadFile(filename)
	_, _ = w.client.WriteFile(w.ctx, &pb.WriteFileRequest{Filename: filename, Content: string(fileContent)})
	// unlock file
	_, _ = w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: false})
	return nil
}

// 必須要件 open, close, read, write
func (w *ClientWrapper) Close(file *os.File) error {
	// ローカルファイルをclose
	err := file.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}
	// リモートファイルclose
	filename := file.Name()
	_, _ = w.client.CloseFile(w.ctx, &pb.CloseFileRequest{Filename: filename})
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (w *ClientWrapper) Write(file *os.File, buf []byte) (int, error) {
	return file.Write(buf)
}

func (w *ClientWrapper) Read(file *os.File, buf []byte) (int, error) {
	return file.Read(buf)
}

// 最低要件open
func (w *ClientWrapper) Open(filename string, mode string) (*os.File, error) {
	// check lock
	var file *os.File
	var err error
	switch mode {
	case "r":
		// open read は lock されていても開ける
		if fileExists(filename) {
			file, err = w.OpenAsReadWithCache(filename)
		} else {
			file, err = w.OpenAsReadWithoutCache(filename)
		}
	case "w":
		// open write は lock されていたら開けない
		locked, _ := w.client.CheckLock(w.ctx, &pb.CheckLockRequest{Filename: filename})
		if locked.GetLocked() {
			return nil, fmt.Errorf("file is locked")
		}
		if fileExists(filename) {
			file, err = w.OpenAsWriteWithCache(filename)
		} else {
			file, err = w.OpenAsWriteWithoutCache(filename)
		}
	default:
		return nil, fmt.Errorf("invalid mode")
	}
	return file, err
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func main() {
	conn, err := grpc.Dial(clientName, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewDFSClient(conn) // c.Hoge()
	ctx := context.Background()
	w := NewClientWrapper(c, ctx)

	// input file name
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter file name:")
	scanner.Scan()
	filename := scanner.Text()
	// input mode
	fmt.Println("Enter mode: r/w")
	scanner.Scan()
	mode := scanner.Text()
	// open file
	file, err := w.Open(filename, mode)
	if err != nil {
		log.Fatalf("could not open file: %v", err)
	}

	// read/write file
	var bytes int
	fileinfo, err := file.Stat()
	if err != nil {
		log.Fatalf("could not get file info: %v", err)
	}
	if mode == "r" {
		filesize := fileinfo.Size()
		buf := make([]byte, filesize)
		bytes, err = w.Read(file, buf)
		if err != nil {
			log.Fatalf("could not read: %v", err)
		}
		log.Printf("Read response: %d", bytes)
		log.Printf("File content: %s", string(buf))
	} else if mode == "w" {
		fmt.Println("Enter file content:")
		scanner.Scan()
		content := scanner.Text()
		bytes, err = w.Write(file, []byte(content))
		if err != nil {
			log.Fatalf("could not write: %v", err)
		}
		log.Printf("Write response: %d", bytes)
		log.Printf("File content: %s", content)
		err := w.FinalizeWrite(file)
		if err != nil {
			log.Fatalf("could not finalize write: %v", err)
		}
	}
	// close file
	err = w.Close(file)
	if err != nil {
		log.Fatalf("could not close file: %v", err)
	}
}

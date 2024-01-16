package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	pb "mygrpc/pkg/grpc"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

var (
	port       int
	host       string
	clientName string
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

func (w *ClientWrapper) OpenAsReadWithoutCache(filename, uuid string) (*os.File, error) {
	fileResponse, err := w.client.OpenFile(w.ctx, &pb.OpenFileRequest{Filename: filename}) // w.clinet.Hoge()
	if err != nil {
		return nil, err
	}
	if _, err := w.client.UpdateCache(w.ctx, &pb.UpdateCacheRequest{Filename: filename, Uuid: uuid}); err != nil {
		return nil, err
	}
	// create file
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	// write file
	if err := os.WriteFile(filename, []byte(fileResponse.GetContent()), 0644); err != nil {
		return nil, err
	}
	fmt.Printf("create cache: %s\n", filename)
	return file, nil
}

func (w *ClientWrapper) OpenAsReadWithCache(filename string) (*os.File, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (w *ClientWrapper) OpenAsWriteWithoutCache(filename string) (*os.File, error) {
	// lock file
	if _, err := w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true}); err != nil {
		return nil, err
	}
	fmt.Printf("send invalid: %s\n", filename)
	// send invalid from server to other client, other clinet delete cache
	// req := &pb.InvalidNotificationRequest{Filename: filename}
	// stream, err := w.client.InvalidNotification(w.ctx, req)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }
	// var invalid *pb.InvalidNotificationResponse
	// if stream != nil {
	// 	for {
	// 		invalid, err = stream.Recv()
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		// delete file
	// 		if invalid != nil && invalid.GetInvalid() {
	// 			fmt.Println("delete file: ", filename)
	// 			os.Remove(filename)
	// 			break
	// 		}
	// 	}
	// }
	// delete cache
	if _, err := w.client.DeleteCache(w.ctx, &pb.DeleteCacheRequest{Filename: filename}); err != nil {
		return nil, err
	}
	// create file
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (w *ClientWrapper) OpenAsWriteWithCache(filename string) (*os.File, error) {
	// lock file
	if _, err := w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true}); err != nil {
		return nil, err
	}
	fmt.Printf("send invalid: %s\n", filename)
	// send invalid from server to other client, other clinet delete cache
	// req := &pb.InvalidNotificationRequest{Filename: filename, Except: &wrapperspb.StringValue{Value: clientName}}
	// stream, err := w.client.InvalidNotification(w.ctx, req)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return nil, err
	// }
	// var invalid *pb.InvalidNotificationResponse
	// if stream != nil {
	// 	for {
	// 		invalid, err = stream.Recv()
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 	delete file
	// 	if invalid != nil && invalid.GetInvalid() {
	// 		fmt.Println("delete file: ", filename)
	// 		os.Remove(filename)
	// 		break
	// 	}
	// }
	// }
	// delete cache
	if _, err := w.client.DeleteCache(w.ctx, &pb.DeleteCacheRequest{Filename: filename}); err != nil {
		return nil, err
	}
	// create file
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (w *ClientWrapper) FinalizeWrite(file *os.File, uuid string) error {
	filename := file.Name()
	// send file content to server
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	if _, err := w.client.WriteFile(w.ctx, &pb.WriteFileRequest{Filename: filename, Content: string(fileContent)}); err != nil {
		return err
	}
	// unlock file
	if _, err := w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: false}); err != nil {
		return err
	}
	// update cache
	if _, err = w.client.UpdateCache(w.ctx, &pb.UpdateCacheRequest{Filename: filename, Uuid: uuid}); err != nil {
		return err
	}
	return nil
}

// 必須要件 open, close, read, write
func (w *ClientWrapper) Close(file *os.File) error {
	// ローカルファイルをclose
	if err := file.Close(); err != nil {
		return err
	}
	// リモートファイルclose
	filename := file.Name()
	if _, err := w.client.CloseFile(w.ctx, &pb.CloseFileRequest{Filename: filename}); err != nil {
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
func (w *ClientWrapper) Open(filename, mode, uuid string) (*os.File, error) {
	// check lock
	var file *os.File
	var err error
	switch mode {
	case "r":
		// open read は lock されていても開ける
		if fileExists(filename) {
			file, err = w.OpenAsReadWithCache(filename)
			if err != nil {
				return nil, err
			}
		} else {
			file, err = w.OpenAsReadWithoutCache(filename, uuid)
			if err != nil {
				return nil, err
			}
		}
	case "w":
		// open write は lock されていたら開けない
		locked, err := w.client.CheckLock(w.ctx, &pb.CheckLockRequest{Filename: filename})
		if err != nil {
			return nil, err
		}
		if locked.GetLocked() {
			return nil, fmt.Errorf("file is locked")
		}
		if fileExists(filename) {
			file, err = w.OpenAsWriteWithCache(filename)
			if err != nil {
				return nil, err
			}
		} else {
			file, err = w.OpenAsWriteWithoutCache(filename)
			if err != nil {
				return nil, err
			}
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

func loadConfig(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("error: %v", err)
	}

	port = config.Port
	host = config.Host
	clientName = fmt.Sprintf("%s:%d", host, port)
}

func main() {
	loadConfig("config.yaml") // load config
	fmt.Println("port: ", port)

	// uuidを生成
	uuid, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("failed to generate uuid: %v", err)
	}
	// stringに変換
	uuidString := uuid.String()

	conn, err := grpc.Dial(clientName, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock()) // grpc connection
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	fmt.Println("connected to ", clientName)
	c := pb.NewDFSClient(conn) // c.Hoge()
	ctx := context.Background()
	w := NewClientWrapper(c, ctx)

	stream, err := c.InvalidNotification(ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			// ファイル名の入力を求める
			fmt.Println("Enter file name:")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			filename := scanner.Text()

			// モードの入力を求める
			fmt.Println("Enter mode: r/w")
			scanner.Scan()
			mode := scanner.Text()
			
			// ファイルを開く
			file, err := w.Open(filename, mode, uuidString)
			if err != nil {
				log.Printf("could not open file: %v", err)
				continue // エラーが発生した場合、次のイテレーションへ
			}

			// read/write file
			fileinfo, err := file.Stat()
			if err != nil {
				log.Fatalf("could not get file info: %v", err)
			}
			if mode == "r" {
				filesize := fileinfo.Size()
				buf := make([]byte, filesize)
				bytes, err := w.Read(file, buf)
				if err != nil {
					log.Fatalf("could not read: %v", err)
				}
				log.Printf("Read response: %d", bytes)
				log.Printf("File content: %s", string(buf))
			} else if mode == "w" {
				fmt.Println("Enter file content:")
				scanner.Scan()
				content := scanner.Text()
				bytes, err := w.Write(file, []byte(content))
				if err != nil {
					log.Fatalf("could not write: %v", err)
				}
				log.Printf("Write response: %d", bytes)
				log.Printf("File content: %s", content)
				if err := w.FinalizeWrite(file, uuidString); err != nil {
					log.Fatalf("could not finalize write: %v", err)
				}
				if err := stream.Send(&pb.InvalidNotificationRequest{Filename: filename, Uid: uuidString}); err != nil {
					log.Fatal(err)
				}
				log.Printf("send invalid: %s\n", filename)
			}
			// close file
			if err := w.Close(file); err != nil {
				log.Fatalf("could not close file: %v", err)
			}
		}
	}() // goroutine
		
	for {
		res, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("recv: %s", res.String())
	}
}

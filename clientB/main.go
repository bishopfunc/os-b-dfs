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

func (w *ClientWrapper) OpenAsReadWithoutCache(filepath, filename, uuid string) (*os.File, error) {
	fileResponse, err := w.client.OpenFile(w.ctx, &pb.OpenFileRequest{Filename: filename})
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

func (w *ClientWrapper) OpenAsWriteWithoutCache(filepath, filename string) (*os.File, error) {
	// lock file
	if filepath != "" {
		_, err := w.client.CreateDir(w.ctx, &pb.CreateDirRequest{Filepath: filepath})
		if err != nil {
			return nil, err
		}
	}
	if _, err := w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true}); err != nil {
		fmt.Println("failed to update lock:")
		return nil, err
	}
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
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (w *ClientWrapper) FinalizeWrite(filename, uuid string) error {
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
func (w *ClientWrapper) Open(filepath, filename, mode, uuid string) (*os.File, error) {
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
			file, err = w.OpenAsReadWithoutCache(filepath, filename, uuid)
			if err != nil {
				return nil, err
			}
		}
	case "w":
		// open write は lock されていたら開けない
		locked, err := w.client.CheckLock(w.ctx, &pb.CheckLockRequest{Filename: filename})
		if err != nil {
			fmt.Println("check lock failed")
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
			file, err = w.OpenAsWriteWithoutCache(filepath, filename)
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

	stream, err := c.NotifyInvalid(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// 一度呼んでおかないとreadだけしたクライアントがclientServersMapに追加されないため、偽のリクエストを送る
	if err := stream.Send(&pb.NotifyInvalidRequest{Filename: "", Uid: uuidString}); err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			// ファイルが存在するディレクトリの入力を求める
			fmt.Println("Enter directory where the file exists:")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			filepath := scanner.Text()

			// ファイル名の入力を求める
			fmt.Println("Enter file name:")
			scanner.Scan()
			filename := scanner.Text()

			// ディレクトリ名が空でなければフォルダfilepathを作る
			// filenameをfilepath/filenameに更新する
			if filepath != "" {
				err := os.MkdirAll(filepath, 0755)
				if err != nil {
					fmt.Println("failed to make directory:", err)
					continue
				}
				filename = filepath + "/" + filename
			}

			// モードの入力を求める
			fmt.Println("Enter mode: r/w")
			scanner.Scan()
			mode := scanner.Text()

			// ファイルを開く
			file, err := w.Open(filepath, filename, mode, uuidString)
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
				if err := stream.Send(&pb.NotifyInvalidRequest{Filename: filename, Uid: uuidString}); err != nil {
					log.Fatal(err)
				}
				if err := w.FinalizeWrite(filename, uuidString); err != nil {
					log.Fatalf("could not finalize write: %v", err)
				}
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
		// ローカルのres.GetFilename()のファイルを削除する
		if err := os.Remove(res.GetFilename()); err != nil {
			log.Fatalf("could not remove file: %v", err)
		}
		// file-serverのhaveCacheUserIDsMap[res.GetFilename()]を削除する
		if _, err := w.client.DeleteCache(w.ctx, &pb.DeleteCacheRequest{Filename: res.GetFilename()}); err != nil {
			log.Fatalf("could not delete cache: %v", err)
		}
	}
}

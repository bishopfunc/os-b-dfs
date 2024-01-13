package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	pb "mygrpc/pkg/grpc"

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
	client          pb.DFSClient
	ctx             context.Context
	cachedFiles     []string // キャッシュしているファイル名のスライス
	lastHealthCheck time.Time
}

func NewClientWrapper(client pb.DFSClient, ctx context.Context, cachedFiles []string, lastHealthCheck time.Time) *ClientWrapper {
	return &ClientWrapper{
		client:          client,
		ctx:             ctx,
		cachedFiles:     cachedFiles,
		lastHealthCheck: lastHealthCheck,
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
	w.cachedFiles = append(w.cachedFiles, filename)
	return file, err
}

func (w *ClientWrapper) OpenAsReadWithCache(filename string) (*os.File, error) {
	file, err := os.Open(filename)
	return file, err
}

func (w *ClientWrapper) OpenAsWriteWithoutCache(filename string) (*os.File, error) {
	// lock file
	_, _ = w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true})
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
	_, _ = w.client.DeleteCache(w.ctx, &pb.DeleteCacheRequest{Filename: filename})
	// create file
	file, err := os.Create(filename)
	w.cachedFiles = append(w.cachedFiles, filename)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return file, err
}

func (w *ClientWrapper) OpenAsWriteWithCache(filename string) (*os.File, error) {
	// lock file
	_, _ = w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: true})
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
	_, _ = w.client.WriteFile(w.ctx, &pb.WriteFileRequest{Filename: filename, Content: string(fileContent), ClientPort: int32(port)})
	// unlock file
	_, _ = w.client.UpdateLock(w.ctx, &pb.UpdateLockRequest{Filename: filename, Lock: false})
	// update cache
	_, _ = w.client.UpdateCache(w.ctx, &pb.UpdateCacheRequest{Filename: filename, Client: clientName})
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

func loadConfig(filename string) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	port = config.Port
	host = config.Host
	clientName = fmt.Sprintf("%s:%d", host, port)
}

func (w *ClientWrapper) DeleteCacheFile(filename string) {
	// cachedFiles スライスから指定されたファイル名を削除する
	for i, file := range w.cachedFiles {
		if file == filename {
			w.cachedFiles = append(w.cachedFiles[:i], w.cachedFiles[i+1:]...)
			os.Remove(filename)
			break
		}
	}
}

func (w *ClientWrapper) HealthCheck() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C // タイマーイベントを待つ

		if len(w.cachedFiles) == 0 {
			// log.Println("No cached files to check")
			continue
		}

		// for _, filename := range w.cachedFiles {
		// 	log.Printf("Cached file: %s", filename)
		// }

		w.lastHealthCheck = time.Now()
		response, err := w.client.HealthCheck(w.ctx, &pb.HealthCheckRequest{FileNames: w.cachedFiles})
		if err != nil {
			log.Printf("Health check failed: %v", err)
			continue
		}

		// サーバーからの応答を確認し、キャッシュされたファイルが変更された場合は削除する
		for _, status := range response.FileStatuses {
			if status.ClientPort != int32(port) && status.UpdatedTime.AsTime().After(w.lastHealthCheck) { // 他のクライアントによって更新された場合
				w.DeleteCacheFile(status.Filename) // キャッシュから削除
				log.Printf("deleted invalid cache file: %s", status.Filename)
			}
		}
	}
}

func main() {
	loadConfig("config.yaml") // load config
	fmt.Println("port: ", port)

	conn, err := grpc.Dial(clientName, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock()) // grpc connection
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	fmt.Println("connected to ", clientName)
	c := pb.NewDFSClient(conn) // c.Hoge()
	ctx := context.Background()
	var cachedFiles []string
	lastHealthCheck := time.Now()
	w := NewClientWrapper(c, ctx, cachedFiles, lastHealthCheck)

	go w.HealthCheck()

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
		file, err := w.Open(filename, mode)
		if err != nil {
			log.Printf("could not open file: %v", err)
			continue // エラーが発生した場合、次のイテレーションへ
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
}
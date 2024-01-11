package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func ReadWithoutCache(filename string) (*os.File, error) {
	return os.Open(filename)
}

func ReadWithCache(filename string) (*os.File, error) {
	return os.Open(filename)
}

func WriteWithoutCache(filename string) (*os.File, error) {
	return os.Open(filename)
}

func WriteWithCache(filename string) (*os.File, error) {
	return os.Open(filename)
}

// 必須要件 open, close, read, write
func Close(fileObj *os.File) error {
	return nil
}

func Write(fileObj *os.File, buf []byte) (int, error) {
	return fileObj.Write(buf)
}

func Read(fileObj *os.File, buf []byte) (int, error) {
	return fileObj.Read(buf)
}

// 最低要件open
func Open(filename string, mode string) (*os.File, error) {
	var file *os.File
	var err error
	switch mode {
	case "r":
		if fileExists(filename) {
			file, err = ReadWithCache(filename)
		} else {
			file, err = ReadWithoutCache(filename)
		}
	case "w":
		if fileExists(filename) {
			file, err = WriteWithCache(filename)
		} else {
			file, err = WriteWithoutCache(filename)
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
	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	// c := pb.NewFileServiceClient(conn)

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second) // 1 second timeout
	// defer cancel()

	// Read file
	// r, err := c.Read(ctx, &pb.FileRequest{Name: "example.txt"})
	// if err != nil {
	// 	log.Fatalf("could not read: %v", err)
	// }
	// log.Printf("File content: %s", r.GetContent())

	// Write file
	// w, err := c.Write(ctx, &pb.FileRequest{Name: "example.txt"})
	// if err != nil {
	// 	log.Fatalf("could not write: %v", err)
	// }
	// log.Printf("Write response: %s", w.GetContent())

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
	file, err := Open(filename, mode)
	if err != nil {
		log.Fatalf("could not open file: %v", err)
	}

	// read/write file
	var buf []byte
	var bytes int
	if mode == "r" {
		bytes, err = Read(file, buf)
		if err != nil {
			log.Fatalf("could not read: %v", err)
		}
		log.Printf("Read response: %d", bytes)
		log.Printf("File content: %s", buf)
	} else if mode == "w" {
		bytes, err = Write(file, buf)
		if err != nil {
			log.Fatalf("could not write: %v", err)
		}
		log.Printf("Write response: %d", bytes)
	}
	// close file
	err = Close(file)
	if err != nil {
		log.Fatalf("could not close file: %v", err)
	}
}

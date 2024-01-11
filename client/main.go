package main

import (
	"context"
	"log"
	"time"

	pb "mygrpc/pkg/grpc"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewFileServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Read file
	r, err := c.Read(ctx, &pb.FileRequest{Name: "example.txt"})
	if err != nil {
		log.Fatalf("could not read: %v", err)
	}
	log.Printf("File content: %s", r.GetContent())

	// Write file
	w, err := c.Write(ctx, &pb.FileRequest{Name: "example.txt"})
	if err != nil {
		log.Fatalf("could not write: %v", err)
	}
	log.Printf("Write response: %s", w.GetContent())
}

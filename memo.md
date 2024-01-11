## 環境構築
```
go mod init mygrpc
go get -u google.golang.org/grpc@v1.50.0
go get -u google.golang.org/protobuf/cmd/protoc-gen-go

mkdir -p pkg/grpc
cd api
protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
dfs.proto
```

## 実行
```
cd file-server
go run main.go

cd client
go run main.go
```
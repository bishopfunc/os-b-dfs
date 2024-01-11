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

## gRPC関連
- `pkg/grpc/`以下は自動生成なので触らない
- `dfs.proto`を編集したら、以下のコマンドで関数を自動生成する
```
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

## 実行例
OpenAsReadWithCache: `client/`に`abc.txt`がある
```
yamamotoshoutanoMacBook-Air:client bishop$ go run main.go 
Enter file name:
abc.txt
Enter mode: r/w
r
2024/01/12 07:30:00 Read response: 7
2024/01/12 07:30:00 File content: abcdefg
```

OpenAsReadWithoutCache: `file-server/`に`a.txt`があり, `client`に何もない
```
yamamotoshoutanoMacBook-Air:client bishop$ go run main.go 
Enter file name:
a.txt
Enter mode: r/w
r
create cache: a.txt
2024/01/12 07:24:14 Read response: 1
```

## TODO
client/main.go
- [ ] OpenAsWriteWithCache
- [ ] OpenAsWriteWithoutCache


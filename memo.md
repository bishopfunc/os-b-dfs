## 環境構築
```
go mod init mygrpc
go get -u google.golang.org/grpc@v1.50.0
go get -u google.golang.org/protobuf/cmd/protoc-gen-go
go get -u gopkg.in/yaml.v2

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

## 注意
- clientA, Bは同じコード、分散システムをイメージして別フォルダにしてる
- 最初の図からちょっと変更ある
- 途中まで書いたコードが残ってるけど、gRPCだとブロードキャスト通信ができないっぽいので困ってる。つまり、ファイルサーバにあるファイルを書き換えてもキャッシュを無効化できない(=invalidの送信ができない)


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
### client/main.go
- [x] OpenAsReadWithCache
- [x] OpenAsReadWithoutCache
- [x] OpenAsWriteWithCache
- [x] OpenAsWriteWithoutCache
- [x] Open
- [x] Close
- [x] Read
- [x] Write
- [x] deleteFile(直接削除すればいいから関数としてはいらない)
- [x] requestLock,requestUnLockも関数としてはいらないかも？


### file-server/main.go
- [x] checkLock
- [x] UpdateLock
- [ ] invalidの送信(多分サーバーストリーミング？)
- [ ] sendInvalid
- [x] UpdateCache
- [x] OpenFile
- [x] deleteCache



cd client
go mod init client

go get google.golang.org/grpc
go get google.golang.org/protobuf/cmd/protoc-gen-go
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc


cd file-server
go mod init file-server
go get google.golang.org/grpc
go get google.golang.org/protobuf/cmd/protoc-gen-go
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc

go mod init mygrpc
go get -u google.golang.org/grpc@v1.50.0
go get -u google.golang.org/protobuf/cmd/protoc-gen-go

mkdir -p pkg/grpc
cd api
protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
dfs.proto
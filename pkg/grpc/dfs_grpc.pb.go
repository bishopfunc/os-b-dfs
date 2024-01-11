// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v4.25.1
// source: dfs.proto

package grpc

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DFSClient is the client API for DFS service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DFSClient interface {
	Read(ctx context.Context, in *FileRequest, opts ...grpc.CallOption) (*FileResponse, error)
	Write(ctx context.Context, in *FileRequest, opts ...grpc.CallOption) (*FileResponse, error)
	OpenFile(ctx context.Context, in *OpenFileRequest, opts ...grpc.CallOption) (*OpenFileResponse, error)
	UpdateCache(ctx context.Context, in *UpdateCacheRequest, opts ...grpc.CallOption) (*UpdateCacheResponse, error)
}

type dFSClient struct {
	cc grpc.ClientConnInterface
}

func NewDFSClient(cc grpc.ClientConnInterface) DFSClient {
	return &dFSClient{cc}
}

func (c *dFSClient) Read(ctx context.Context, in *FileRequest, opts ...grpc.CallOption) (*FileResponse, error) {
	out := new(FileResponse)
	err := c.cc.Invoke(ctx, "/dfs.DFS/Read", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dFSClient) Write(ctx context.Context, in *FileRequest, opts ...grpc.CallOption) (*FileResponse, error) {
	out := new(FileResponse)
	err := c.cc.Invoke(ctx, "/dfs.DFS/Write", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dFSClient) OpenFile(ctx context.Context, in *OpenFileRequest, opts ...grpc.CallOption) (*OpenFileResponse, error) {
	out := new(OpenFileResponse)
	err := c.cc.Invoke(ctx, "/dfs.DFS/OpenFile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dFSClient) UpdateCache(ctx context.Context, in *UpdateCacheRequest, opts ...grpc.CallOption) (*UpdateCacheResponse, error) {
	out := new(UpdateCacheResponse)
	err := c.cc.Invoke(ctx, "/dfs.DFS/UpdateCache", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DFSServer is the server API for DFS service.
// All implementations must embed UnimplementedDFSServer
// for forward compatibility
type DFSServer interface {
	Read(context.Context, *FileRequest) (*FileResponse, error)
	Write(context.Context, *FileRequest) (*FileResponse, error)
	OpenFile(context.Context, *OpenFileRequest) (*OpenFileResponse, error)
	UpdateCache(context.Context, *UpdateCacheRequest) (*UpdateCacheResponse, error)
	mustEmbedUnimplementedDFSServer()
}

// UnimplementedDFSServer must be embedded to have forward compatible implementations.
type UnimplementedDFSServer struct {
}

func (UnimplementedDFSServer) Read(context.Context, *FileRequest) (*FileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Read not implemented")
}
func (UnimplementedDFSServer) Write(context.Context, *FileRequest) (*FileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Write not implemented")
}
func (UnimplementedDFSServer) OpenFile(context.Context, *OpenFileRequest) (*OpenFileResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method OpenFile not implemented")
}
func (UnimplementedDFSServer) UpdateCache(context.Context, *UpdateCacheRequest) (*UpdateCacheResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateCache not implemented")
}
func (UnimplementedDFSServer) mustEmbedUnimplementedDFSServer() {}

// UnsafeDFSServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DFSServer will
// result in compilation errors.
type UnsafeDFSServer interface {
	mustEmbedUnimplementedDFSServer()
}

func RegisterDFSServer(s grpc.ServiceRegistrar, srv DFSServer) {
	s.RegisterService(&DFS_ServiceDesc, srv)
}

func _DFS_Read_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DFSServer).Read(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dfs.DFS/Read",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DFSServer).Read(ctx, req.(*FileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DFS_Write_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DFSServer).Write(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dfs.DFS/Write",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DFSServer).Write(ctx, req.(*FileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DFS_OpenFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(OpenFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DFSServer).OpenFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dfs.DFS/OpenFile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DFSServer).OpenFile(ctx, req.(*OpenFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DFS_UpdateCache_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateCacheRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DFSServer).UpdateCache(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/dfs.DFS/UpdateCache",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DFSServer).UpdateCache(ctx, req.(*UpdateCacheRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DFS_ServiceDesc is the grpc.ServiceDesc for DFS service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DFS_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "dfs.DFS",
	HandlerType: (*DFSServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Read",
			Handler:    _DFS_Read_Handler,
		},
		{
			MethodName: "Write",
			Handler:    _DFS_Write_Handler,
		},
		{
			MethodName: "OpenFile",
			Handler:    _DFS_OpenFile_Handler,
		},
		{
			MethodName: "UpdateCache",
			Handler:    _DFS_UpdateCache_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "dfs.proto",
}

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: logic/room.ext.proto

package logicpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	RoomExtService_PushRoom_FullMethodName = "/logic.RoomExtService/PushRoom"
)

// RoomExtServiceClient is the client API for RoomExtService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RoomExtServiceClient interface {
	// 推送消息到房间
	PushRoom(ctx context.Context, in *PushRoomRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type roomExtServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewRoomExtServiceClient(cc grpc.ClientConnInterface) RoomExtServiceClient {
	return &roomExtServiceClient{cc}
}

func (c *roomExtServiceClient) PushRoom(ctx context.Context, in *PushRoomRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, RoomExtService_PushRoom_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RoomExtServiceServer is the server API for RoomExtService service.
// All implementations must embed UnimplementedRoomExtServiceServer
// for forward compatibility.
type RoomExtServiceServer interface {
	// 推送消息到房间
	PushRoom(context.Context, *PushRoomRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedRoomExtServiceServer()
}

// UnimplementedRoomExtServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedRoomExtServiceServer struct{}

func (UnimplementedRoomExtServiceServer) PushRoom(context.Context, *PushRoomRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PushRoom not implemented")
}
func (UnimplementedRoomExtServiceServer) mustEmbedUnimplementedRoomExtServiceServer() {}
func (UnimplementedRoomExtServiceServer) testEmbeddedByValue()                        {}

// UnsafeRoomExtServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RoomExtServiceServer will
// result in compilation errors.
type UnsafeRoomExtServiceServer interface {
	mustEmbedUnimplementedRoomExtServiceServer()
}

func RegisterRoomExtServiceServer(s grpc.ServiceRegistrar, srv RoomExtServiceServer) {
	// If the following call pancis, it indicates UnimplementedRoomExtServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&RoomExtService_ServiceDesc, srv)
}

func _RoomExtService_PushRoom_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PushRoomRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RoomExtServiceServer).PushRoom(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: RoomExtService_PushRoom_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RoomExtServiceServer).PushRoom(ctx, req.(*PushRoomRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RoomExtService_ServiceDesc is the grpc.ServiceDesc for RoomExtService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RoomExtService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "logic.RoomExtService",
	HandlerType: (*RoomExtServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PushRoom",
			Handler:    _RoomExtService_PushRoom_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "logic/room.ext.proto",
}

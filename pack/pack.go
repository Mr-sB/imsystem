package pack

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"imsystem/protopack"
)

var (
	ErrUnknownProtoType = fmt.Errorf("unknown proto type: %w", protopack.ErrProtoPack)
	ErrUnknownOpType    = fmt.Errorf("unknown op type: %w", protopack.ErrProtoPack)
	ErrUnknownPushType  = fmt.Errorf("unknown push type: %w", protopack.ErrProtoPack)

	bodyCreatorRouter = map[pb.ProtoType]func(*pb.HeadPack) (proto.Message, error){
		pb.ProtoType_PROTO_TYPE_REQUEST:  createRequest,
		pb.ProtoType_PROTO_TYPE_RESPONSE: createResponse,
		pb.ProtoType_PROTO_TYPE_PUSH:     createPush,
	}
	reqMessageCreator = map[pb.OpType]func() proto.Message{
		pb.OpType_OP_TYPE_HEARTBEAT:    func() proto.Message { return new(pb.HeartbeatReq) },
		pb.OpType_OP_TYPE_BROADCAST:    func() proto.Message { return new(pb.BroadcastReq) },
		pb.OpType_OP_TYPE_QUERY:        func() proto.Message { return new(pb.QueryReq) },
		pb.OpType_OP_TYPE_RENAME:       func() proto.Message { return new(pb.RenameReq) },
		pb.OpType_OP_TYPE_PRIVATE_CHAT: func() proto.Message { return new(pb.PrivateChatReq) },
	}
	rspMessageCreator = map[pb.OpType]func() proto.Message{
		pb.OpType_OP_TYPE_HEARTBEAT:    func() proto.Message { return new(pb.HeartbeatRsp) },
		pb.OpType_OP_TYPE_BROADCAST:    func() proto.Message { return new(pb.BroadcastRsp) },
		pb.OpType_OP_TYPE_QUERY:        func() proto.Message { return new(pb.QueryRsp) },
		pb.OpType_OP_TYPE_RENAME:       func() proto.Message { return new(pb.RenameRsp) },
		pb.OpType_OP_TYPE_PRIVATE_CHAT: func() proto.Message { return new(pb.PrivateChatRsp) },
	}
	pushMessageCreator = map[pb.PushType]func() proto.Message{
		pb.PushType_PUSH_TYPE_KICK:         func() proto.Message { return new(pb.KickPush) },
		pb.PushType_PUSH_TYPE_BROADCAST:    func() proto.Message { return new(pb.BroadcastPush) },
		pb.PushType_PUSH_TYPE_PRIVATE_CHAT: func() proto.Message { return new(pb.PrivateChatPush) },
	}
)

type Packer struct {
	*protopack.ProtoPacker
}

func NewPacker() *Packer {
	return &Packer{
		ProtoPacker: protopack.NewProtoPacker(bodyCreator),
	}
}

////////////////////

func bodyCreator(head *pb.HeadPack) (proto.Message, error) {
	router, ok := bodyCreatorRouter[head.ProtoType]
	if !ok {
		return nil, ErrUnknownProtoType
	}
	return router(head)
}

func createRequest(head *pb.HeadPack) (proto.Message, error) {
	creator, ok := reqMessageCreator[pb.OpType(head.Type)]
	if !ok {
		return nil, ErrUnknownOpType
	}
	return creator(), nil
}

func createResponse(head *pb.HeadPack) (proto.Message, error) {
	creator, ok := rspMessageCreator[pb.OpType(head.Type)]
	if !ok {
		return nil, ErrUnknownOpType
	}
	return creator(), nil
}

func createPush(head *pb.HeadPack) (proto.Message, error) {
	creator, ok := pushMessageCreator[pb.PushType(head.Type)]
	if !ok {
		return nil, ErrUnknownPushType
	}
	return creator(), nil
}

////////////////////

func NewRequestHead(pid uint32, opType pb.OpType) *pb.HeadPack {
	return &pb.HeadPack{
		ProtoType: pb.ProtoType_PROTO_TYPE_REQUEST,
		Pid:       pid,
		Type:      uint32(opType),
	}
}

func NewResponseHead(pid uint32, opType pb.OpType, code pb.ResponseCode) *pb.HeadPack {
	return &pb.HeadPack{
		ProtoType: pb.ProtoType_PROTO_TYPE_RESPONSE,
		Pid:       pid,
		Type:      uint32(opType),
		Code:      uint32(code),
	}
}

func NewPushHead(pushType pb.PushType) *pb.HeadPack {
	return &pb.HeadPack{
		ProtoType: pb.ProtoType_PROTO_TYPE_PUSH,
		Type:      uint32(pushType),
	}
}

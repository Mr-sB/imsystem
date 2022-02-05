package pack

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"imsystem/protopack"
	"io"
)

var (
	ErrUnknownProtoType = fmt.Errorf("unknown proto type: %w", protopack.ErrProtoPack)
	ErrUnknownOpType    = fmt.Errorf("unknown op type: %w", protopack.ErrProtoPack)
	ErrUnknownPushType  = fmt.Errorf("unknown push type: %w", protopack.ErrProtoPack)

	unmarshalBodyRouter = map[pb.ProtoType]func(*pb.HeadPack, []byte) (proto.Message, error){
		pb.ProtoType_PROTO_TYPE_REQUEST:  unmarshalRequest,
		pb.ProtoType_PROTO_TYPE_RESPONSE: unmarshalResponse,
		pb.ProtoType_PROTO_TYPE_PUSH:     unmarshalPush,
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

func Encode(head *pb.HeadPack, body proto.Message) ([]byte, error) {
	return protopack.Encode(head, body)
}

func Decode(reader io.Reader) (*pb.HeadPack, proto.Message, error) {
	head, bodyBytes, err := protopack.Decode(reader)
	if err != nil {
		return head, nil, err
	}
	//解析消息体
	var body proto.Message = nil
	router, ok := unmarshalBodyRouter[head.ProtoType]
	if !ok {
		err = ErrUnknownProtoType
	} else {
		body, err = router(head, bodyBytes)
	}
	return head, body, err
}

////////////////////

func unmarshalRequest(head *pb.HeadPack, bytes []byte) (proto.Message, error) {
	creator, ok := reqMessageCreator[pb.OpType(head.Type)]
	if !ok {
		return nil, ErrUnknownOpType
	}
	message := creator()
	err := proto.Unmarshal(bytes, message)
	return message, err
}

func unmarshalResponse(head *pb.HeadPack, bytes []byte) (proto.Message, error) {
	creator, ok := rspMessageCreator[pb.OpType(head.Type)]
	if !ok {
		return nil, ErrUnknownOpType
	}
	message := creator()
	err := proto.Unmarshal(bytes, message)
	return message, err
}

func unmarshalPush(head *pb.HeadPack, bytes []byte) (proto.Message, error) {
	creator, ok := pushMessageCreator[pb.PushType(head.Type)]
	if !ok {
		return nil, ErrUnknownOpType
	}
	message := creator()
	err := proto.Unmarshal(bytes, message)
	return message, err
}

////////////////////

func NewRequestHead(pid uint32, opType pb.OpType) *pb.HeadPack {
	return &pb.HeadPack{
		ProtoType: pb.ProtoType_PROTO_TYPE_REQUEST,
		Pid:       pid,
		Type:      uint32(opType),
	}
}

func NewResponseHead(pid uint32, opType pb.OpType, code uint32) *pb.HeadPack {
	return &pb.HeadPack{
		ProtoType: pb.ProtoType_PROTO_TYPE_RESPONSE,
		Pid:       pid,
		Type:      uint32(opType),
		Code:      code,
	}
}

func NewPushHead(pushType pb.PushType) *pb.HeadPack {
	return &pb.HeadPack{
		ProtoType: pb.ProtoType_PROTO_TYPE_PUSH,
		Type:      uint32(pushType),
	}
}

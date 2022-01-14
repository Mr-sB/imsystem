package protopack

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"imsystem/pack"
	"imsystem/pb"
	"io"
)

var (
	ErrUnknownProtoType = errors.New("proto pack: unknown proto type")
	ErrUnknownOpType    = errors.New("proto pack: unknown op type")
	ErrUnknownPushType  = errors.New("proto pack: unknown push type")
)

func Encode(message proto.Message) ([]byte, error) {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return bytes, err
	}
	return pack.Encode(bytes), nil
}

func Decode(reader io.Reader) (*pb.NetPacketBase, proto.Message, proto.Message, error) {
	bytes, err := pack.Decode(reader)
	if err != nil {
		return nil, nil, nil, err
	}
	packetBase := &pb.NetPacketBase{}
	err = proto.Unmarshal(bytes, packetBase)
	if err != nil {
		return packetBase, nil, nil, err
	}

	var protoBase, message proto.Message = nil, nil

	switch packetBase.Packet.ProtoType {
	case pb.ProtoType_PROTO_TYPE_UNKNOWN:
		err = ErrUnknownProtoType
	case pb.ProtoType_PROTO_TYPE_REQUEST:
		requestBase := &pb.NetRequestBase{}
		protoBase = requestBase
		err = proto.Unmarshal(bytes, requestBase)
		if err != nil {
			break
		}
		//解析request
		message, err = requestUnmarshal(requestBase, bytes)
	case pb.ProtoType_PROTO_TYPE_RESPONSE:
		responseBase := &pb.NetResponseBase{}
		protoBase = responseBase
		err = proto.Unmarshal(bytes, responseBase)
		if err != nil {
			break
		}
		//解析response
		message, err = responseUnmarshal(responseBase, bytes)
	case pb.ProtoType_PROTO_TYPE_PUSH:
		pushBase := &pb.NetPushBase{}
		protoBase = pushBase
		err = proto.Unmarshal(bytes, pushBase)
		if err != nil {
			break
		}
		//解析push
		message, err = pushUnmarshal(pushBase, bytes)
	}
	return packetBase, protoBase, message, err
}

////////////////////

func requestUnmarshal(requestBase *pb.NetRequestBase, bytes []byte) (proto.Message, error) {
	var message proto.Message = nil
	var err error = nil
	switch requestBase.Request.OpType {
	case pb.OpType_OP_TYPE_UNKNOWN:
		err = ErrUnknownOpType
	case pb.OpType_OP_TYPE_HEARTBEAT:
		//TODO
	case pb.OpType_OP_TYPE_BROADCAST:
		message = &pb.BroadcastReq{}
	case pb.OpType_OP_TYPE_QUERY:
		message = &pb.QueryReq{}
	case pb.OpType_OP_TYPE_RENAME:
		message = &pb.RenameReq{}
	case pb.OpType_OP_TYPE_PRIVATE_CHAT:
		message = &pb.PrivateChatReq{}
	}
	if err != nil {
		return message, err
	}
	if message == nil {
		return nil, ErrUnknownOpType
	}
	err = proto.Unmarshal(bytes, message)
	return message, err
}

func responseUnmarshal(responseBase *pb.NetResponseBase, bytes []byte) (proto.Message, error) {
	var message proto.Message = nil
	var err error = nil
	switch responseBase.Response.OpType {
	case pb.OpType_OP_TYPE_UNKNOWN:
		err = ErrUnknownOpType
	case pb.OpType_OP_TYPE_HEARTBEAT:
		//TODO
	case pb.OpType_OP_TYPE_BROADCAST:
		message = &pb.BroadcastRsp{}
	case pb.OpType_OP_TYPE_QUERY:
		message = &pb.QueryRsp{}
	case pb.OpType_OP_TYPE_RENAME:
		message = &pb.RenameRsp{}
	case pb.OpType_OP_TYPE_PRIVATE_CHAT:
		message = &pb.PrivateChatRsp{}
	}
	if err != nil {
		return message, err
	}
	if message == nil {
		return nil, ErrUnknownOpType
	}
	err = proto.Unmarshal(bytes, message)
	return message, err
}

func pushUnmarshal(pushBase *pb.NetPushBase, bytes []byte) (proto.Message, error) {
	var message proto.Message = nil
	var err error = nil
	switch pushBase.Push.PushType {
	case pb.PushType_PUSH_TYPE_UNKNOWN:
		err = ErrUnknownPushType
	case pb.PushType_PUSH_TYPE_KICK:
		message = &pb.KickPush{}
	case pb.PushType_PUSH_TYPE_BROADCAST:
		message = &pb.BroadcastPush{}
	case pb.PushType_PUSH_TYPE_PRIVATE_CHAT:
		message = &pb.PrivateChatPush{}
	}
	if err != nil {
		return message, err
	}
	if message == nil {
		return nil, ErrUnknownPushType
	}
	err = proto.Unmarshal(bytes, message)
	return message, err
}

////////////////////

func NewNetPacket(protoType pb.ProtoType) *pb.NetPacket {
	return &pb.NetPacket{
		ProtoType: protoType,
	}
}

func NewNetRequestPacket() *pb.NetPacket {
	return NewNetPacket(pb.ProtoType_PROTO_TYPE_REQUEST)
}

func NewNetResponsePacket() *pb.NetPacket {
	return NewNetPacket(pb.ProtoType_PROTO_TYPE_RESPONSE)
}

func NewNetPushPacket() *pb.NetPacket {
	return NewNetPacket(pb.ProtoType_PROTO_TYPE_PUSH)
}

func NewNetRequest(opType pb.OpType) *pb.NetRequest {
	return &pb.NetRequest{
		OpType: opType,
	}
}

func NewNetResponse(opType pb.OpType, code int32, msg string) *pb.NetResponse {
	return &pb.NetResponse{
		OpType: opType,
		Code:   code,
		Msg:    msg,
	}
}

func NewNetPush(pushType pb.PushType) *pb.NetPush {
	return &pb.NetPush{
		PushType: pushType,
	}
}

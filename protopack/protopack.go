package protopack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"io"
)

const (
	HeadLen  = 2 //比TotalLen小
	TotalLen = 4
)

var (
	ErrProtoPack        = errors.New("proto pack error")
	ErrUnknownProtoType = fmt.Errorf("unknown proto type: %w", ErrProtoPack)
	ErrUnknownOpType    = fmt.Errorf("unknown op type: %w", ErrProtoPack)
	ErrUnknownPushType  = fmt.Errorf("unknown push type: %w", ErrProtoPack)
)

func Encode(head *pb.HeadPack, body proto.Message) ([]byte, error) {
	headBytes, err := proto.Marshal(head)
	if err != nil {
		return headBytes, err
	}
	bodyBytes, err := proto.Marshal(body)
	if err != nil {
		return bodyBytes, err
	}
	//头部长度
	headLen := uint16(len(headBytes))
	//总长度
	totalLen := uint32(TotalLen + HeadLen + int(headLen) + len(bodyBytes))

	message := make([]byte, totalLen)
	//写入总长度
	binary.LittleEndian.PutUint32(message, totalLen)
	//写入头部长度
	binary.LittleEndian.PutUint16(message[TotalLen:], headLen)
	//复制头部
	copy(message[TotalLen+HeadLen:], headBytes)
	//复制消息体
	copy(message[TotalLen+HeadLen+headLen:], bodyBytes)
	return message, nil
}

func Decode(reader io.Reader) (*pb.HeadPack, proto.Message, error) {
	//读取消息长度
	bytes := make([]byte, TotalLen)
	_, err := io.ReadFull(reader, bytes)
	if err != nil {
		return nil, nil, err
	}
	totalLen := binary.LittleEndian.Uint32(bytes)
	//读取消息头部长度
	_, err = io.ReadFull(reader, bytes[:HeadLen])
	if err != nil {
		return nil, nil, err
	}
	headLen := binary.LittleEndian.Uint16(bytes)
	//读取头部
	headBytes := make([]byte, headLen)
	_, err = io.ReadFull(reader, headBytes)
	if err != nil {
		return nil, nil, err
	}
	head := &pb.HeadPack{}
	err = proto.Unmarshal(headBytes, head)
	if err != nil {
		return head, nil, err
	}
	//读取消息体
	bodyBytes := make([]byte, totalLen-TotalLen-HeadLen-uint32(headLen))
	_, err = io.ReadFull(reader, bodyBytes)
	if err != nil {
		return nil, nil, err
	}
	//解析消息体
	var body proto.Message = nil
	switch head.ProtoType {
	case pb.ProtoType_PROTO_TYPE_UNKNOWN:
		err = ErrUnknownProtoType
	case pb.ProtoType_PROTO_TYPE_REQUEST:
		//解析request
		body, err = requestUnmarshal(head, bodyBytes)
	case pb.ProtoType_PROTO_TYPE_RESPONSE:
		//解析response
		body, err = responseUnmarshal(head, bodyBytes)
	case pb.ProtoType_PROTO_TYPE_PUSH:
		//解析push
		body, err = pushUnmarshal(head, bodyBytes)
	}
	return head, body, err
}

////////////////////

func requestUnmarshal(head *pb.HeadPack, bytes []byte) (proto.Message, error) {
	var message proto.Message = nil
	var err error = nil
	switch pb.OpType(head.Type) {
	case pb.OpType_OP_TYPE_UNKNOWN:
		err = ErrUnknownOpType
	case pb.OpType_OP_TYPE_HEARTBEAT:
		message = &pb.HeartbeatReq{}
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

func responseUnmarshal(head *pb.HeadPack, bytes []byte) (proto.Message, error) {
	var message proto.Message = nil
	var err error = nil
	switch pb.OpType(head.Type) {
	case pb.OpType_OP_TYPE_UNKNOWN:
		err = ErrUnknownOpType
	case pb.OpType_OP_TYPE_HEARTBEAT:
		message = &pb.HeartbeatRsp{}
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

func pushUnmarshal(head *pb.HeadPack, bytes []byte) (proto.Message, error) {
	var message proto.Message = nil
	var err error = nil
	switch pb.PushType(head.Type) {
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

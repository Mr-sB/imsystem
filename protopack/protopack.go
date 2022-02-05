package protopack

import (
	"encoding/binary"
	"errors"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"io"
)

const (
	HeadLen  = 2 //不大于TotalLen
	TotalLen = 4
)

var (
	ErrProtoPack        = errors.New("proto pack error")
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

func Decode(reader io.Reader) (*pb.HeadPack, []byte, error) {
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
	head := new(pb.HeadPack)
	err = proto.Unmarshal(headBytes, head)
	if err != nil {
		return head, nil, err
	}
	//读取消息体
	bodyBytes := make([]byte, totalLen-TotalLen-HeadLen-uint32(headLen))
	_, err = io.ReadFull(reader, bodyBytes)
	return head, bodyBytes, err
}

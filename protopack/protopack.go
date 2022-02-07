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
	HeadLen  = 2 //不大于TotalLen
	TotalLen = 4
)

var (
	ErrProtoPack      = errors.New("proto pack error")
	ErrNilBodyCreator = fmt.Errorf("unknown op type: %w", ErrProtoPack)
)

type ProtoPacker struct {
	bodyCreator func(*pb.HeadPack) (proto.Message, error)
	buffer      []byte
}

func NewProtoPacker(creator func(*pb.HeadPack) (proto.Message, error)) *ProtoPacker {
	return &ProtoPacker{
		bodyCreator: creator,
		buffer:      make([]byte, 1024),
	}
}

func (p *ProtoPacker) Encode(head *pb.HeadPack, body proto.Message) ([]byte, error) {
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

func (p *ProtoPacker) Decode(reader io.Reader) (*pb.HeadPack, proto.Message, error) {
	//读取消息长度
	p.buffer = ensures(p.buffer, TotalLen)
	totalLenBytes := p.buffer[:TotalLen]
	_, err := io.ReadFull(reader, totalLenBytes)
	if err != nil {
		return nil, nil, err
	}
	totalLen := binary.LittleEndian.Uint32(totalLenBytes)
	//读取消息头部长度
	p.buffer = ensures(p.buffer, TotalLen+HeadLen)
	headLenBytes := p.buffer[TotalLen : TotalLen+HeadLen]
	_, err = io.ReadFull(reader, headLenBytes)
	if err != nil {
		return nil, nil, err
	}
	headLen := binary.LittleEndian.Uint16(headLenBytes)
	//读取头部
	p.buffer = ensures(p.buffer, TotalLen+HeadLen+int(headLen))
	headBytes := p.buffer[TotalLen+HeadLen : TotalLen+HeadLen+headLen]
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
	p.buffer = ensures(p.buffer, int(totalLen))
	bodyBytes := p.buffer[TotalLen+HeadLen+headLen : totalLen]
	_, err = io.ReadFull(reader, bodyBytes)
	if err != nil {
		return head, nil, err
	}
	if p.bodyCreator == nil {
		return head, nil, ErrNilBodyCreator
	}
	body, err := p.bodyCreator(head)
	if err != nil {
		return head, nil, err
	}
	err = proto.Unmarshal(bodyBytes, body)
	return head, body, err
}

func ensures(bytes []byte, length int) []byte {
	curLen := len(bytes)
	if curLen >= length {
		return bytes
	}
	twoTimesLen := 2 * curLen
	var newSize int
	if length < twoTimesLen {
		newSize = twoTimesLen
	} else {
		newSize = length
	}
	newBytes := make([]byte, newSize)
	copy(newBytes, bytes)
	return newBytes
}

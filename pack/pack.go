package pack

import (
	"encoding/binary"
	"errors"
	"io"
)

const HeadLen = 4

var (
	ErrNilMessage  = errors.New("pack: nil message")
)

//TODO 对象池化slice，减少开销
//TODO slice对象池可以按照一定的块大小做分层。例如8字节为单位区分不同的slice池

func Encode(message []byte) []byte {
	//消息长度
	messageLen := uint32(len(message))
	//包体
	pkg := make([]byte, messageLen + HeadLen)
	//写入消息长度
	binary.LittleEndian.PutUint32(pkg, messageLen)
	//复制消息内容
	copy(pkg[HeadLen:], message)
	return pkg
}

func EncodeString(message string) []byte {
	return Encode([]byte(message))
}

func Decode(reader io.Reader) ([]byte, error) {
	//读取消息长度
	head := make([]byte, HeadLen)
	_, err := io.ReadFull(reader, head)
	if err != nil {
		return nil, err
	}
	messageLen := binary.LittleEndian.Uint32(head)
	//读取消息
	message := make([]byte, messageLen)
	_, err = io.ReadFull(reader, message)
	//返回消息内容
	return message, err
}

func DecodeString(reader io.Reader) (string, error) {
	message, err := Decode(reader)
	if err != nil {
		return "", err
	}
	if message == nil {
		return "", ErrNilMessage
	}
	return string(message), nil
}

package imserver

import (
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	_ "google.golang.org/protobuf/proto"
	"imsystem/pb"
	_ "imsystem/pb"
	"imsystem/protopack"
	"net"
	"sync/atomic"
	"time"
)

type User struct {
	Name            string
	Addr            string
	conn            net.Conn
	messageChan     chan []byte
	aliveChan       chan struct{}
	closeChan       chan struct{}
	online          uint32
	serverInterface ServerInterface
	pid             uint32
}

func NewUser(conn net.Conn, serverInterface ServerInterface) *User {
	addr := conn.RemoteAddr().String()
	user := &User{
		Name:            addr,
		Addr:            addr,
		conn:            conn,
		messageChan:     make(chan []byte),
		aliveChan:       make(chan struct{}),
		closeChan:       make(chan struct{}),
		online:          0,
		serverInterface: serverInterface,
	}
	return user
}

func (u *User) String() string {
	return fmt.Sprintf("用户[%s] Addr[%s]", u.Name, u.Addr)
}

func (u *User) Online() {
	u.setOnline(true)
	go u.listenMessage()

	//监听用户消息
	go u.readRemote()

	//监听是否超时
	go u.listenTimeout()
}

//写到channel里等待取出发送
func (u *User) WriteMessage(message []byte) {
	u.messageChan <- message
}

//直接发送消息
func (u *User) SendMessage(message []byte) {
	_, err := u.conn.Write(message)
	if err != nil {
		fmt.Println("conn write error:", err, u)
		//下线
		u.offline()
	}
}

func (u *User) IsOnline() bool {
	return atomic.LoadUint32(&u.online) != 0
}

func (u *User) setOnline(online bool) {
	var value uint32
	if online {
		value = 1
	} else {
		value = 0
	}
	atomic.StoreUint32(&u.online, value)
}

func (u *User) offline() {
	//避免重复offline
	if !u.IsOnline() {
		return
	}
	u.setOnline(false)

	//让所有的go程结束
	close(u.closeChan)
	u.conn.Close()
	u.serverInterface.UserOffline(u)
}

func (u *User) readRemote() {
	for {
		packetBase, protoBase, message, err := protopack.Decode(u.conn)
		if errors.Is(err, protopack.ErrUnknownProtoType) || errors.Is(err, protopack.ErrUnknownOpType) || errors.Is(err, protopack.ErrUnknownPushType) {
			fmt.Println("conn read error:", err, u)
			continue
		}
		if err != nil {
			fmt.Println("conn read error, offline:", err, u)
			//下线
			u.offline()
			return
		}
		u.aliveChan <- struct{}{}
		u.requestHandler(packetBase, protoBase, message)
	}
}

func (u *User) listenTimeout() {
	for {
		select {
		case <-u.closeChan:
			return
		case <-u.aliveChan:
			//什么都不做，只是唤醒等待，刷新所有case求值，重置timer
			//优先级select  优先响应close
			if u.isChanClosed() {
				return
			}
		case <-time.After(3 * time.Minute):
			//优先级select  优先响应close
			if u.isChanClosed() {
				return
			}
			//超时被踢
			//Close conn之后Read会error，从而触发offline
			bytes, _ := protopack.Encode(&pb.KickPush{
				Packet:   protopack.NewNetPushPacket(),
				Push:     protopack.NewNetPush(pb.PushType_PUSH_TYPE_KICK),
			})
			u.SendMessage(bytes)
			u.conn.Close()
		}
	}
}

func (u *User) requestHandler(packerBase *pb.NetPacketBase, protoBase proto.Message, message proto.Message) {
	if packerBase.Packet.ProtoType != pb.ProtoType_PROTO_TYPE_REQUEST {
		return
	}
	requestBase, ok := protoBase.(*pb.NetRequestBase)
	if !ok {
		return
	}

	switch requestBase.Request.OpType {
	case pb.OpType_OP_TYPE_HEARTBEAT:
		if !ok {
			return
		}
		bytes, _ := protopack.Encode(&pb.HeartbeatRsp{
			Packet:   protopack.NewNetResponsePacket(),
			Response: protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_HEARTBEAT, 200, ""),
		})
		//response
		u.SendMessage(bytes)
	case pb.OpType_OP_TYPE_BROADCAST:
		request, ok := message.(*pb.BroadcastReq)
		if !ok {
			return
		}
		bytes, _ := protopack.Encode(&pb.BroadcastRsp{
			Packet:   protopack.NewNetResponsePacket(),
			Response: protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_BROADCAST, 200, ""),
		})
		//response
		u.SendMessage(bytes)
		//push
		u.broadcast(&pb.BroadcastPush{
			Packet:  protopack.NewNetPushPacket(),
			Push:    protopack.NewNetPush(pb.PushType_PUSH_TYPE_BROADCAST),
			User:    u.String(),
			Content: request.Content,
		})
	case pb.OpType_OP_TYPE_QUERY:
		bytes, _ := protopack.Encode(&pb.QueryRsp{
			Packet:   protopack.NewNetResponsePacket(),
			Response: protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_QUERY, 200, ""),
			Users:    u.serverInterface.Query(),
		})
		//response
		u.SendMessage(bytes)
	case pb.OpType_OP_TYPE_RENAME:
		request, ok := message.(*pb.RenameReq)
		if !ok {
			return
		}
		ok = u.serverInterface.Rename(u, request.NewName)

		response := &pb.RenameRsp{
			Packet: protopack.NewNetResponsePacket(),
		}
		if ok {
			response.Response = protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_RENAME, 200, "")
			response.NewName = request.NewName
		} else {
			response.Response = protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_RENAME, 500, "改名失败，名字重复："+request.NewName)
		}
		bytes, _ := protopack.Encode(response)
		//response
		u.SendMessage(bytes)
	case pb.OpType_OP_TYPE_PRIVATE_CHAT:
		request, ok := message.(*pb.PrivateChatReq)
		if !ok {
			return
		}
		err := u.serverInterface.PrivateChat(u, request.User, request.Content)
		response := &pb.PrivateChatRsp{
			Packet: protopack.NewNetResponsePacket(),
		}
		if err == nil {
			response.Response = protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_PRIVATE_CHAT, 200, "")
		} else {
			response.Response = protopack.NewNetResponse(requestBase.Request.Pid, pb.OpType_OP_TYPE_PRIVATE_CHAT, 500, err.Error())
		}
		bytes, _ := protopack.Encode(response)
		//response
		u.SendMessage(bytes)
	}
}

func (u *User) broadcast(broadcastPush *pb.BroadcastPush) {
	u.serverInterface.Broadcast(broadcastPush)
}

func (u *User) listenMessage() {
	for {
		select {
		case <-u.closeChan:
			return
		case message := <-u.messageChan:
			//优先级select  优先响应close
			if u.isChanClosed() {
				return
			}
			u.conn.Write(message)
		}
	}
}

func (u *User) isChanClosed() bool {
	select {
	case <-u.closeChan:
		return true
	default:
		return false
	}
}

func (u *User) getPid() uint32 {
	return atomic.AddUint32(&u.pid, 1)
}

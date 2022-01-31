package imserver

import (
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
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
	requestRouter   map[pb.OpType]func(*pb.HeadPack, proto.Message)
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
	user.initRouter()
	return user
}

func (u *User) initRouter() {
	u.requestRouter = make(map[pb.OpType]func(*pb.HeadPack, proto.Message), 5)
	u.requestRouter[pb.OpType_OP_TYPE_HEARTBEAT] = u.reqHeartbeat
	u.requestRouter[pb.OpType_OP_TYPE_BROADCAST] = u.reqBroadcast
	u.requestRouter[pb.OpType_OP_TYPE_QUERY] = u.reqQuery
	u.requestRouter[pb.OpType_OP_TYPE_RENAME] = u.reqRename
	u.requestRouter[pb.OpType_OP_TYPE_PRIVATE_CHAT] = u.reqPrivateChat
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
		head, body, err := protopack.Decode(u.conn)
		if errors.Is(err, protopack.ErrProtoPack) {
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
		u.requestHandler(head, body)
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
			bytes, _ := protopack.Encode(protopack.NewPushHead(pb.PushType_PUSH_TYPE_KICK), &pb.KickPush{})
			u.SendMessage(bytes)
			u.conn.Close()
		}
	}
}

func (u *User) requestHandler(reqHead *pb.HeadPack, reqBody proto.Message) {
	if reqHead.ProtoType != pb.ProtoType_PROTO_TYPE_REQUEST {
		return
	}
	router, ok := u.requestRouter[pb.OpType(reqHead.Type)]
	if !ok || router == nil {
		return
	}
	router(reqHead, reqBody)
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

//Router
func (u *User) reqHeartbeat(reqHead *pb.HeadPack, reqBody proto.Message) {
	bytes, _ := protopack.Encode(
		protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_HEARTBEAT, pb.ResponseCodeSuccess),
		&pb.HeartbeatRsp{})
	//response
	u.SendMessage(bytes)
}

func (u *User) reqBroadcast(reqHead *pb.HeadPack, reqBody proto.Message) {
	request, ok := reqBody.(*pb.BroadcastReq)
	if !ok {
		return
	}
	bytes, _ := protopack.Encode(
		protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_BROADCAST, pb.ResponseCodeSuccess),
		&pb.BroadcastRsp{})
	//response
	u.SendMessage(bytes)
	//push
	u.broadcast(&pb.BroadcastPush{
		User:    u.String(),
		Content: request.Content,
	})
}

func (u *User) reqQuery(reqHead *pb.HeadPack, reqBody proto.Message) {
	bytes, _ := protopack.Encode(
		protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_QUERY, pb.ResponseCodeSuccess),
		&pb.QueryRsp{
			Users: u.serverInterface.Query(),
		})
	//response
	u.SendMessage(bytes)
}

func (u *User) reqRename(reqHead *pb.HeadPack, reqBody proto.Message) {
	request, ok := reqBody.(*pb.RenameReq)
	if !ok {
		return
	}
	ok = u.serverInterface.Rename(u, request.NewName)

	var head *pb.HeadPack
	body := &pb.RenameRsp{}
	if ok {
		head = protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_RENAME, pb.ResponseCodeSuccess)
		body.NewName = request.NewName
	} else {
		head = protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_RENAME, pb.RenameError)
	}
	bytes, _ := protopack.Encode(head, body)
	//response
	u.SendMessage(bytes)
}

func (u *User) reqPrivateChat(reqHead *pb.HeadPack, reqBody proto.Message) {
	request, ok := reqBody.(*pb.PrivateChatReq)
	if !ok {
		return
	}
	err := u.serverInterface.PrivateChat(u, request.User, request.Content)
	var head *pb.HeadPack
	body := &pb.PrivateChatRsp{}
	if err == nil {
		head = protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_PRIVATE_CHAT, pb.ResponseCodeSuccess)
	} else {
		head = protopack.NewResponseHead(reqHead.Pid, pb.OpType_OP_TYPE_PRIVATE_CHAT, pb.ChatUserError)
	}
	bytes, _ := protopack.Encode(head, body)
	//response
	u.SendMessage(bytes)
}

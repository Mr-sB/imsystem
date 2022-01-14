package imclient

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"imsystem/protopack"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Client struct {
	Ip         string
	Port       int
	Name       string
	conn       net.Conn
	closeChan  chan struct{}
	online     bool
	onlineLock sync.RWMutex
	pid        int32
	pidLock    sync.Mutex //没必要读写锁
}

func (c *Client) String() string {
	return fmt.Sprintf("用户[%s] Addr[%s]", c.Name, net.JoinHostPort(c.Ip, strconv.Itoa(c.Port)))
}

func NewClient(ip string, port int) *Client {
	return &Client{
		Ip:        ip,
		Port:      port,
		closeChan: make(chan struct{}),
		online:    false,
	}
}

func (c *Client) Start() bool {
	conn, err := net.Dial("tcp", net.JoinHostPort(c.Ip, strconv.Itoa(c.Port)))
	if err != nil {
		fmt.Println("connect error:", err, c)
		return false
	}
	c.conn = conn
	c.Name = net.JoinHostPort(c.Ip, strconv.Itoa(c.Port))
	c.online = true
	//开始处理逻辑
	//go 接收
	go c.readRemote()

	return true
}

func (c *Client) SendMessage(message []byte) {
	c.onlineLock.RLock()
	if !c.online {
		fmt.Println("send error: client is offline!")
		return
	}
	c.onlineLock.RUnlock()
	_, err := c.conn.Write(message)
	if err != nil {
		fmt.Println("send error:", err)
	}
}

func (c *Client) readRemote() {
	for {
		packetBase, protoBase, message, err := protopack.Decode(c.conn)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			fmt.Println("conn read error, offline:", err, c)
			//下线
			c.offline()
			//err == io.EOF 合法下线
			return
		} else if err != nil {
			fmt.Println("conn read error:", err, c)
			continue
		}
		c.handler(packetBase, protoBase, message)
	}
}

func (c *Client) offline() {
	c.onlineLock.Lock()
	if !c.online {
		c.onlineLock.Unlock()
		return
	}
	c.online = false
	c.onlineLock.Unlock()

	//让所有的go程结束
	close(c.closeChan)
	c.conn.Close()
}

func (c *Client) getPid() int32 {
	c.pidLock.Lock()
	defer c.pidLock.Unlock()
	c.pid++
	return c.pid
}

func (c *Client) SendBroadcast(req *pb.BroadcastReq) {
	req.Packet = protopack.NewNetRequestPacket(c.getPid())
	req.Request = protopack.NewNetRequest(pb.OpType_OP_TYPE_BROADCAST)
	bytes, _ := protopack.Encode(req)
	c.SendMessage(bytes)
}

func (c *Client) SendQuery(req *pb.QueryReq) {
	req.Packet = protopack.NewNetRequestPacket(c.getPid())
	req.Request = protopack.NewNetRequest(pb.OpType_OP_TYPE_QUERY)
	bytes, _ := protopack.Encode(req)
	c.SendMessage(bytes)
}

func (c *Client) SendRename(req *pb.RenameReq) {
	req.Packet = protopack.NewNetRequestPacket(c.getPid())
	req.Request = protopack.NewNetRequest(pb.OpType_OP_TYPE_RENAME)
	bytes, _ := protopack.Encode(req)
	c.SendMessage(bytes)
}

func (c *Client) SendPrivateChat(req *pb.PrivateChatReq) {
	req.Packet = protopack.NewNetRequestPacket(c.getPid())
	req.Request = protopack.NewNetRequest(pb.OpType_OP_TYPE_PRIVATE_CHAT)
	bytes, _ := protopack.Encode(req)
	c.SendMessage(bytes)
}

func (c *Client) handler(packerBase *pb.NetPacketBase, protoBase proto.Message, message proto.Message) {
	switch packerBase.Packet.ProtoType {
	case pb.ProtoType_PROTO_TYPE_RESPONSE:
		responseBase, ok := protoBase.(*pb.NetResponseBase)
		if !ok {
			return
		}
		c.responseHandler(responseBase, message)
	case pb.ProtoType_PROTO_TYPE_PUSH:
		pushBase, ok := protoBase.(*pb.NetPushBase)
		if !ok {
			return
		}
		c.pushHandler(pushBase, message)
	}
}

func (c *Client) responseHandler(responseBase *pb.NetResponseBase, message proto.Message) {
	if responseBase.Response.Code != 200 {
		fmt.Println("request failed:", responseBase.Response)
	}
	switch responseBase.Response.OpType {
	case pb.OpType_OP_TYPE_HEARTBEAT:
		//TODO
	case pb.OpType_OP_TYPE_BROADCAST:

	case pb.OpType_OP_TYPE_QUERY:
		if responseBase.Response.Code != 200 {
			return
		}
		response, ok := message.(*pb.QueryRsp)
		if !ok {
			return
		}
		fmt.Println(strings.Join(response.Users, "\n"))
	case pb.OpType_OP_TYPE_RENAME:
		if responseBase.Response.Code != 200 {
			return
		}
		response, ok := message.(*pb.RenameRsp)
		if !ok {
			return
		}
		fmt.Println("New name:", response.NewName)
	case pb.OpType_OP_TYPE_PRIVATE_CHAT:

	}
}

func (c *Client) pushHandler(pushBase *pb.NetPushBase, message proto.Message) {
	switch pushBase.Push.PushType {
	case pb.PushType_PUSH_TYPE_KICK:
		fmt.Println("kicked by server")
	case pb.PushType_PUSH_TYPE_BROADCAST:
		push, ok := message.(*pb.BroadcastPush)
		if !ok {
			return
		}
		fmt.Println("广播:", push.User, push.Content)
	case pb.PushType_PUSH_TYPE_PRIVATE_CHAT:
		push, ok := message.(*pb.PrivateChatPush)
		if !ok {
			return
		}
		fmt.Println("私聊:", push.User, push.Content)
	}
}

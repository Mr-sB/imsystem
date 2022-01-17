package imclient

import (
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"imsystem/protopack"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	HEARTBEAT_INTERVAL          = 5 * time.Second
	HEARTBEAT_MAX_TIMEOUT_COUNT = 3
)

type Client struct {
	Ip             string
	Port           int
	Name           string
	conn           net.Conn
	closeChan      chan struct{}
	online         uint32 //0:false else:true
	pid            uint32
	hbTimeoutCount uint32
	closeWait      sync.WaitGroup
}

func (c *Client) String() string {
	return fmt.Sprintf("用户[%s] Addr[%s]", c.Name, net.JoinHostPort(c.Ip, strconv.Itoa(c.Port)))
}

func NewClient(ip string, port int) *Client {
	return &Client{
		Ip:     ip,
		Port:   port,
		online: 0,
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
	c.closeChan = make(chan struct{})
	c.setOnline(true)
	atomic.StoreUint32(&c.hbTimeoutCount, 0)
	//开始处理逻辑
	//go 接收
	c.closeWait.Add(2)
	go c.readRemote()
	go c.keepConnection()

	return true
}

func (c *Client) sendMessage(message proto.Message) {
	bytes, err := protopack.Encode(message)
	if err != nil {
		fmt.Println("send error:", err)
	}
	if !c.IsOnline() {
		fmt.Println("send error: client is Disconnect!")
		return
	}
	_, err = c.conn.Write(bytes)
	if err != nil {
		fmt.Println("send error:", err)
	}
}

func (c *Client) readRemote() {
	defer c.closeWait.Done()
	for {
		packetBase, protoBase, message, err := protopack.Decode(c.conn)
		if errors.Is(err, protopack.ErrUnknownProtoType) || errors.Is(err, protopack.ErrUnknownOpType) || errors.Is(err, protopack.ErrUnknownPushType) {
			fmt.Println("conn read error:", err, c)
			continue
		}
		if err != nil {
			fmt.Println("conn read error, Disconnect:", err, c)
			//下线
			c.Disconnect()
			//err == io.EOF 合法下线
			return
		}
		c.handler(packetBase, protoBase, message)
	}
}

//发送心跳包保持连接，超时的时候自动重连
func (c *Client) keepConnection() {
	ticker := time.NewTicker(HEARTBEAT_INTERVAL)

	close := func(){
		c.closeWait.Done()
		ticker.Stop()
	}

	for {
		select {
		case <-c.closeChan:
			close()
			return
		case <-ticker.C:
			if c.isChanClosed(){
				close()
				return
			}
			hbTimeoutCount := atomic.AddUint32(&c.hbTimeoutCount, 1)
			if hbTimeoutCount > HEARTBEAT_MAX_TIMEOUT_COUNT {
				close()
				//心跳超时
				c.Reconnect()
				return
			}
			//时间到，发送心跳包
			c.SendHeartbeat(&pb.HeartbeatReq{})
		}
	}
}

func (c *Client) Disconnect() {
	if !c.IsOnline() {
		return
	}
	c.setOnline(false)

	//让所有的go程结束
	close(c.closeChan)
	c.conn.Close()
	c.closeWait.Wait()
}

func (c *Client) Reconnect() {
	fmt.Println("Start reconnect.", c)
	c.Disconnect()
	success := c.Start()
	fmt.Println("Reconnect result:", success, c.IsOnline(), c)
}

func (c *Client) IsOnline() bool {
	return atomic.LoadUint32(&c.online) != 0
}

func (c *Client) setOnline(online bool) {
	var value uint32
	if online {
		value = 1
	} else {
		value = 0
	}
	atomic.StoreUint32(&c.online, value)
}

func (c *Client) isChanClosed() bool {
	select {
	case <-c.closeChan:
		return true
	default:
		return false
	}
}

func (c *Client) getPid() uint32 {
	return atomic.AddUint32(&c.pid, 1)
}

func (c *Client) SendHeartbeat(req *pb.HeartbeatReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = protopack.NewNetRequest(c.getPid(), pb.OpType_OP_TYPE_HEARTBEAT)
	c.sendMessage(req)
}

func (c *Client) SendBroadcast(req *pb.BroadcastReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = protopack.NewNetRequest(c.getPid(), pb.OpType_OP_TYPE_BROADCAST)
	c.sendMessage(req)
}

func (c *Client) SendQuery(req *pb.QueryReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = protopack.NewNetRequest(c.getPid(), pb.OpType_OP_TYPE_QUERY)
	c.sendMessage(req)
}

func (c *Client) SendRename(req *pb.RenameReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = protopack.NewNetRequest(c.getPid(), pb.OpType_OP_TYPE_RENAME)
	c.sendMessage(req)
}

func (c *Client) SendPrivateChat(req *pb.PrivateChatReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = protopack.NewNetRequest(c.getPid(), pb.OpType_OP_TYPE_PRIVATE_CHAT)
	c.sendMessage(req)
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
		//收到心跳包，重置心跳包超时次数
		atomic.StoreUint32(&c.hbTimeoutCount, 0)
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

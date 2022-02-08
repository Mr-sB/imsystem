package imclient

import (
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pack"
	"imsystem/pb"
	"imsystem/protopack"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	HeartbeatInterval        = 5 * time.Second
	HeartbeatMaxTimeoutCount = 3
)

type Client struct {
	Ip             string
	Port           int
	Name           string
	conn           net.Conn
	closeChan      chan struct{}
	online         bool
	onlineLock     sync.RWMutex
	pid            uint32
	hbTimeoutCount uint32
	closeWait      sync.WaitGroup
	responseRouter map[pb.OpType]func(*pb.HeadPack, proto.Message)
	pushRouter     map[pb.PushType]func(proto.Message)
	packer         *pack.Packer
}

func (c *Client) String() string {
	return fmt.Sprintf("用户[%s] Addr[%s]", c.Name, net.JoinHostPort(c.Ip, strconv.Itoa(c.Port)))
}

func NewClient(ip string, port int) *Client {
	return &Client{
		Ip:     ip,
		Port:   port,
		online: false,
		packer: pack.NewPacker(),
	}
}

func (c *Client) InitRouter(responseRouter map[pb.OpType]func(*pb.HeadPack, proto.Message), pushRouter map[pb.PushType]func(proto.Message)) {
	c.responseRouter = responseRouter
	c.pushRouter = pushRouter
}

func (c *Client) Connect() bool {
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

func (c *Client) Disconnect() {
	c.onlineLock.Lock()
	if !c.online{
		c.onlineLock.Unlock()
		return
	}
	c.online = false
	c.onlineLock.Unlock()

	//让所有的go程结束
	close(c.closeChan)
	c.conn.Close()
	c.closeWait.Wait()
}

func (c *Client) Reconnect() {
	fmt.Println("Start reconnect.", c)
	c.Disconnect()
	success := c.Connect()
	fmt.Println("Reconnect result:", success, c)
}

func (c *Client) IsOnline() bool {
	c.onlineLock.RLock()
	defer c.onlineLock.RUnlock()
	return c.online
}

func (c *Client) SendMessage(head *pb.HeadPack, body proto.Message) {
	bytes, err := c.packer.Encode(head, body)
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

func (c *Client) NewRequestHead(opType pb.OpType) *pb.HeadPack {
	return pack.NewRequestHead(c.getPid(), opType)
}

func (c *Client) readRemote() {
	defer c.closeWait.Done()
	for {
		head, body, err := c.packer.Decode(c.conn)
		if errors.Is(err, protopack.ErrProtoPack) {
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
		c.handler(head, body)
	}
}

//发送心跳包保持连接，超时的时候自动重连
func (c *Client) keepConnection() {
	ticker := time.NewTicker(HeartbeatInterval)

	close := func() {
		c.closeWait.Done()
		ticker.Stop()
	}

	for {
		select {
		case <-c.closeChan:
			close()
			return
		case <-ticker.C:
			if c.isChanClosed() {
				close()
				return
			}
			hbTimeoutCount := atomic.AddUint32(&c.hbTimeoutCount, 1)
			if hbTimeoutCount > HeartbeatMaxTimeoutCount {
				close()
				//心跳超时
				c.Reconnect()
				return
			}
			//时间到，发送心跳包
			c.sendHeartbeat()
		}
	}
}

func (c *Client) setOnline(online bool) {
	c.onlineLock.Lock()
	c.online = online
	c.onlineLock.Unlock()
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

func (c *Client) handler(head *pb.HeadPack, body proto.Message) {
	switch head.ProtoType {
	case pb.ProtoType_PROTO_TYPE_RESPONSE:
		c.responseHandler(head, body)
	case pb.ProtoType_PROTO_TYPE_PUSH:
		c.pushHandler(head, body)
	}
}

func (c *Client) responseHandler(head *pb.HeadPack, body proto.Message) {
	if head.Code != pb.ResponseCodeSuccess {
		fmt.Println("request failed:", head)
	}
	opType := pb.OpType(head.Type)
	if opType == pb.OpType_OP_TYPE_HEARTBEAT {
		c.rspHeartbeat()
	} else {
		router, ok := c.responseRouter[opType]
		if !ok || router == nil {
			return
		}
		router(head, body)
	}
}

func (c *Client) pushHandler(head *pb.HeadPack, body proto.Message) {
	router, ok := c.pushRouter[pb.PushType(head.Type)]
	if !ok || router == nil {
		return
	}
	router(body)
}

//Response router
func (c *Client) rspHeartbeat() {
	//收到心跳包，重置心跳包超时次数
	atomic.StoreUint32(&c.hbTimeoutCount, 0)
}

//Send
func (c *Client) sendHeartbeat() {
	c.SendMessage(c.NewRequestHead(pb.OpType_OP_TYPE_HEARTBEAT), &pb.HeartbeatReq{})
}

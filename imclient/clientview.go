package imclient

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"imsystem/protopack"
	"strings"
)

type ClientView struct {
	client         *Client
	responseRouter map[pb.OpType]func(*pb.NetResponseBase, proto.Message)
	pushRouter     map[pb.PushType]func(*pb.NetPushBase, proto.Message)
}

func (c *ClientView) String() string {
	return c.client.String()
}

func NewClientView(client *Client) *ClientView {
	clientView := &ClientView{
		client:         client,
	}
	clientView.initRouter()
	return clientView
}

func(c *ClientView) initRouter() {
	c.responseRouter = make(map[pb.OpType]func(*pb.NetResponseBase, proto.Message), 2)
	c.pushRouter = make(map[pb.PushType]func(*pb.NetPushBase, proto.Message), 3)

	c.responseRouter[pb.OpType_OP_TYPE_QUERY] = rspQuery
	c.responseRouter[pb.OpType_OP_TYPE_RENAME] = rspRename

	c.pushRouter[pb.PushType_PUSH_TYPE_KICK] = pushKick
	c.pushRouter[pb.PushType_PUSH_TYPE_BROADCAST] = pushBroadcast
	c.pushRouter[pb.PushType_PUSH_TYPE_PRIVATE_CHAT] = pushPrivateChat

	c.client.InitRouter(c.responseRouter, c.pushRouter)
}

func (c *ClientView) Start() bool {
	if !c.client.Connect(){
		fmt.Println("连接失败，请重连")
	}

	fmt.Println("1.广播")
	fmt.Println("2.查询在线用户")
	fmt.Println("3.改名")
	fmt.Println("4.私聊")
	fmt.Println("5.重连")
	fmt.Println("99.退出")

	var mode int
	for{
		fmt.Println("请输入模式:")
		fmt.Scanln(&mode)
		switch mode {
		case 1:
			fmt.Println("请输入广播内容:")
			var content string
			fmt.Scanln(&content)
			c.SendBroadcast(&pb.BroadcastReq{
				Content: content,
			})
		case 2:
			c.SendQuery(&pb.QueryReq{})
		case 3:
			fmt.Println("请输入新名字:")
			var newName string
			fmt.Scanln(&newName)
			c.SendRename(&pb.RenameReq{
				NewName: newName,
			})
		case 4:
			fmt.Println("请输入名字和内容:")
			var user, content string
			fmt.Scanln(&user, &content)
			c.SendPrivateChat(&pb.PrivateChatReq{
				User: user,
				Content: content,
			})
		case 5:
			c.client.Reconnect()
		case 99:
			break
		}
	}
}

//Send
func (c *ClientView) SendHeartbeat(req *pb.HeartbeatReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = c.client.NewNetRequest(pb.OpType_OP_TYPE_HEARTBEAT)
	c.client.SendMessage(req)
}

func (c *ClientView) SendBroadcast(req *pb.BroadcastReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = c.client.NewNetRequest(pb.OpType_OP_TYPE_BROADCAST)
	c.client.SendMessage(req)
}

func (c *ClientView) SendQuery(req *pb.QueryReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = c.client.NewNetRequest(pb.OpType_OP_TYPE_QUERY)
	c.client.SendMessage(req)
}

func (c *ClientView) SendRename(req *pb.RenameReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = c.client.NewNetRequest(pb.OpType_OP_TYPE_RENAME)
	c.client.SendMessage(req)
}

func (c *ClientView) SendPrivateChat(req *pb.PrivateChatReq) {
	req.Packet = protopack.NewNetRequestPacket()
	req.Request = c.client.NewNetRequest(pb.OpType_OP_TYPE_PRIVATE_CHAT)
	c.client.SendMessage(req)
}

//Response router
func rspQuery(responseBase *pb.NetResponseBase, message proto.Message) {
	if responseBase.Response.Code != 200 {
		return
	}
	response, ok := message.(*pb.QueryRsp)
	if !ok {
		return
	}
	fmt.Println(strings.Join(response.Users, "\n"))
}

func rspRename(responseBase *pb.NetResponseBase, message proto.Message) {
	if responseBase.Response.Code != 200 {
		return
	}
	response, ok := message.(*pb.RenameRsp)
	if !ok {
		return
	}
	fmt.Println("New name:", response.NewName)
}

//Push router
func pushKick(pushBase *pb.NetPushBase, message proto.Message) {
	fmt.Println("kicked by server")
}

func pushBroadcast(pushBase *pb.NetPushBase, message proto.Message) {
	push, ok := message.(*pb.BroadcastPush)
	if !ok {
		return
	}
	fmt.Println("广播:", push.User, push.Content)
}

func pushPrivateChat(pushBase *pb.NetPushBase, message proto.Message) {
	push, ok := message.(*pb.PrivateChatPush)
	if !ok {
		return
	}
	fmt.Println("私聊:", push.User, push.Content)
}

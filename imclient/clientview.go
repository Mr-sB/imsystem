package imclient

import (
	"fmt"
	"google.golang.org/protobuf/proto"
	"imsystem/pb"
	"strings"
)

type ClientView struct {
	client         *Client
	responseRouter map[pb.OpType]func(*pb.HeadPack, proto.Message)
	pushRouter     map[pb.PushType]func(proto.Message)
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
	c.responseRouter = make(map[pb.OpType]func(*pb.HeadPack, proto.Message), 2)
	c.pushRouter = make(map[pb.PushType]func(proto.Message), 3)

	c.responseRouter[pb.OpType_OP_TYPE_QUERY] = rspQuery
	c.responseRouter[pb.OpType_OP_TYPE_RENAME] = rspRename

	c.pushRouter[pb.PushType_PUSH_TYPE_KICK] = pushKick
	c.pushRouter[pb.PushType_PUSH_TYPE_BROADCAST] = pushBroadcast
	c.pushRouter[pb.PushType_PUSH_TYPE_PRIVATE_CHAT] = pushPrivateChat

	c.client.InitRouter(c.responseRouter, c.pushRouter)
}

func (c *ClientView) Start() {
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
	mainLoop:
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
			c.client.Disconnect()
			break mainLoop
		}
	}
}

//Send

func (c *ClientView) SendBroadcast(body *pb.BroadcastReq) {
	c.client.SendMessage(c.client.NewRequestHead(pb.OpType_OP_TYPE_BROADCAST), body)
}

func (c *ClientView) SendQuery(body *pb.QueryReq) {
	c.client.SendMessage(c.client.NewRequestHead(pb.OpType_OP_TYPE_QUERY), body)
}

func (c *ClientView) SendRename(body *pb.RenameReq) {
	c.client.SendMessage(c.client.NewRequestHead(pb.OpType_OP_TYPE_RENAME), body)
}

func (c *ClientView) SendPrivateChat(body *pb.PrivateChatReq) {
	c.client.SendMessage(c.client.NewRequestHead(pb.OpType_OP_TYPE_PRIVATE_CHAT), body)
}

//Response router
func rspQuery(head *pb.HeadPack, body proto.Message) {
	if pb.ResponseCode(head.Code) != pb.ResponseCodeSuccess {
		return
	}
	response, ok := body.(*pb.QueryRsp)
	if !ok {
		return
	}
	fmt.Println(strings.Join(response.Users, "\n"))
}

func rspRename(head *pb.HeadPack, body proto.Message) {
	if pb.ResponseCode(head.Code) != pb.ResponseCodeSuccess {
		return
	}
	response, ok := body.(*pb.RenameRsp)
	if !ok {
		return
	}
	fmt.Println("New name:", response.NewName)
}

//Push router
func pushKick(body proto.Message) {
	fmt.Println("kicked by server")
}

func pushBroadcast(body proto.Message) {
	push, ok := body.(*pb.BroadcastPush)
	if !ok {
		return
	}
	fmt.Println("广播:", push.User, push.Content)
}

func pushPrivateChat(body proto.Message) {
	push, ok := body.(*pb.PrivateChatPush)
	if !ok {
		return
	}
	fmt.Println("私聊:", push.User, push.Content)
}

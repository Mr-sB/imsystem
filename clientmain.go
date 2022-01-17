package main

import (
	"flag"
	"fmt"
	"imsystem/imclient"
	"imsystem/pb"
)

var (
	ip   string
	port int
)

func init() {
	flag.StringVar(&ip, "ip", "127.0.0.1", "连接的地址")
	flag.IntVar(&port, "port", 8000, "连接的端口")
}

func main() {
	flag.Parse()
	client := imclient.NewClient(ip, port)
	if !client.Start() {
		return
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
			client.SendBroadcast(&pb.BroadcastReq{
				Content: content,
			})
		case 2:
			client.SendQuery(&pb.QueryReq{})
		case 3:
			fmt.Println("请输入新名字:")
			var newName string
			fmt.Scanln(&newName)
			client.SendRename(&pb.RenameReq{
				NewName: newName,
			})
		case 4:
			fmt.Println("请输入名字和内容:")
			var user, content string
			fmt.Scanln(&user, &content)
			client.SendPrivateChat(&pb.PrivateChatReq{
				User: user,
				Content: content,
			})
		case 5:
			client.Reconnect()
		case 99:
			break
		}
	}
}

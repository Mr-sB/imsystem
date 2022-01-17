package main

import (
	"flag"
	"imsystem/imclient"
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
	clientView := imclient.NewClientView(imclient.NewClient(ip, port))
	clientView.Start()
}

package main

import (
	"flag"
	"imsystem/imclient"
)

func main() {
	ip := flag.String("ip", "127.0.0.1", "监听的地址")
	port := flag.Int("port", 8000, "监听的端口")
	flag.Parse()
	clientView := imclient.NewClientView(imclient.NewClient(*ip, *port))
	clientView.Start()
}

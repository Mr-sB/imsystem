package main

import (
	"flag"
	"imsystem/imserver"
)

func main() {
	ip := flag.String("ip", "127.0.0.1", "监听的地址")
	port := flag.Int("port", 8000, "监听的端口")
	flag.Parse()
	server := imserver.NewServer(*ip, *port)
	server.Start()
}

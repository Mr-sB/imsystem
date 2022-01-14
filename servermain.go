package main

import (
	"imsystem/imserver"
)

func main() {
	server := imserver.NewServer("127.0.0.1", 8000)
	server.Start()
}

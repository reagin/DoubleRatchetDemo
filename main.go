package main

import (
	"flag"

	"github.com/reagin/double_ratchet/chat"
)

var port = flag.String("port", "", "Server Listen Port")

func init() {
	flag.Parse()
}

func main() {
	chat.StartDoubleRatchet(*port)
}

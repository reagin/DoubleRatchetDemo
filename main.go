package main

import (
	"flag"

	"github.com/reagin/double_ratchet/chat"
)

var port = flag.String("port", "", "Listen Port")
var mode = flag.String("mode", "", "Running Mode <client/server>")

func init() {
	flag.Parse()
}

func main() {
	chat.StartDoubleRatchet(*port, *mode)
}

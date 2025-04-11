package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/reagin/double_ratchet/core"
)

var port = flag.String("port", "", "Listen Port")
var mode = flag.String("mode", "", "Running Mode <client/server>")

func init() {
	flag.Parse()
}

func getText() string {
	reader := bufio.NewReader(os.Stdin)
	str, _ := reader.ReadBytes('\n')
	str = bytes.TrimSpace(str)
	return string(str)
}

func main() {
	switch *mode {
	case "server":
		server := core.NewServer("127.0.0.1:" + *port)
		go server.StartServer()
		go func() {
			for {
				fmt.Printf("From Client: %s\n", string(<-server.RecvChannel))
			}
		}()
		for {
			str := getText()
			server.SendChannel <- []byte(str)
		}
	case "client":
		fmt.Printf("Please input remote address: ")
		address := getText()
		client := core.NewClient("127.0.0.1:"+*port, address)
		go client.StartClient()
		go func() {
			for {
				fmt.Printf("From Server: %s\n", string(<-client.RecvChannel))
			}
		}()
		for {
			str := getText()
			client.SendChannel <- []byte(str)
		}
	}
}

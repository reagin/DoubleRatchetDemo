package chat

import "github.com/reagin/double_ratchet/core"

const (
	FileType = iota
	TextType
	ClientMode
	ServerMode
)

var (
	localPort     string
	remotePort    string
	localAddress  string
	listenAddress string
	remoteAddress string
	sendChannel   chan []byte
	recvChannel   chan []byte
)

var client *core.Client
var server *core.Server
var dataList []*Message

var runMode = ServerMode
var isChange = make(chan bool)

type Message struct {
	Type    int
	IsSelf  bool
	Content []byte
}

type FileTrunk struct {
	FileName string
	Content  []byte
}

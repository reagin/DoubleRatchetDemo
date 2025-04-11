package chat

import "github.com/reagin/double_ratchet/core"

var (
	localAddress  string
	remoteAddress string
)

var client *core.Client
var server *core.Server
var dataList []*Message
var isChange = make(chan bool)

const FileType = 0
const TextType = 1

type Message struct {
	Type    int
	IsSelf  bool
	Content []byte
}

type FileTrunk struct {
	FileName string
	Content  []byte
}

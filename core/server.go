package core

import (
	"bufio"
	"log"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-reuseport"
	"github.com/reagin/double_ratchet/utils"
)

type Server struct {
	stopChan     chan bool
	stopOnce     sync.Once
	waitGroug    sync.WaitGroup
	netListener  net.Listener
	LocalAddress string
	SendChannel  chan []byte
	RecvChannel  chan []byte
}

func NewServer(localAddress string) *Server {
	return &Server{
		stopChan:     make(chan bool),
		LocalAddress: localAddress,
		SendChannel:  make(chan []byte, 8),
		RecvChannel:  make(chan []byte, 8),
	}
}

func (server *Server) StartServer() {
	go server.handleServer()
}

func (server *Server) StopServer() {
	server.stopOnce.Do(func() {
		log.Println("🛑 服务器正在关闭...")
		close(server.stopChan)

		if server.netListener != nil {
			server.netListener.Close()
		}
		server.waitGroug.Wait()

		log.Println("✅ 服务器已成功关闭")
	})
}

func (server *Server) handleServer() {
	listener, err := reuseport.Listen("tcp", server.LocalAddress)
	if err != nil {
		log.Printf("❌ 服务端监听端口失败: %s\n", err.Error())
		return
	}
	defer listener.Close()
	server.netListener = listener
	log.Printf("🎉 服务端已开始监听: %s\n", server.LocalAddress)

	for {
		listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		connect, err := listener.Accept()
		if err != nil {
			select {
			case <-server.stopChan:
				log.Println("🛑 服务器收到停止信号，终止监听")
				return
			default:
				continue
			}
		}
		log.Printf("🎉 与客户端 %s 成功建立连接\n", connect.RemoteAddr().String())

		server.waitGroug.Add(1)
		go server.handleConnection(connect)
	}
}

func (server *Server) handleConnection(connect net.Conn) {
	defer connect.Close()
	defer server.waitGroug.Done()

	// 记录双棘轮的状态信息
	ratchetState := utils.NewRatchetState()

	// 使用 bufio 修饰 net.Conn
	reader := bufio.NewReader(connect)
	writer := bufio.NewWriter(connect)

	// 生成 Diffe-Hellman 密钥对
	keyPair := utils.NewDiffeHellmanKeyPair()
	pubKeyBytes, _ := utils.EncodeMessage(keyPair.PublicKey.Bytes())

	// 接收客户端的初始公钥
	remotePubKeyBytes, _ := utils.DecodeMessage(reader)
	remotePubKey := utils.BytesToPublicKey(remotePubKeyBytes)

	// 计算共享密钥
	sharedSecret, _ := keyPair.PrivateKey.ECDH(remotePubKey)
	ratchetState.RootChain = sharedSecret

	// 向客户端发送初始公钥
	if _, err := writer.Write(pubKeyBytes); err != nil {
		log.Printf("🤯 向客户端发送信息失败: %s\n", err.Error())
		return
	}
	writer.Flush()

	// 交换公钥用于后续主动发起请求
	// 生成 Diffe-Hellman 密钥对
	keyPair = utils.NewDiffeHellmanKeyPair()
	pubKeyBytes, _ = utils.EncodeMessage(keyPair.PublicKey.Bytes())

	// 接收客户端的公钥
	remotePubKeyBytes, _ = utils.DecodeMessage(reader)
	remotePubKey = utils.BytesToPublicKey(remotePubKeyBytes)

	// 向客户端发送公钥
	if _, err := writer.Write(pubKeyBytes); err != nil {
		log.Printf("🤯 向客户端发送信息失败: %s\n", err.Error())
		return
	}
	writer.Flush()

	server.waitGroug.Add(2)
	// NOTE: 启动 gorunite 发送信息
	go startSendListener(server.stopChan, &server.waitGroug, server.SendChannel, writer, keyPair, ratchetState, &remotePubKey)
	// NOTE: 启动 gorunite 接收信息
	go startRecvListener(server.stopChan, &server.waitGroug, server.RecvChannel, reader, keyPair, ratchetState, &remotePubKey)

	<-server.stopChan
	log.Printf("🛑 关闭与客户端 %s 的连接\n", connect.RemoteAddr().String())
}

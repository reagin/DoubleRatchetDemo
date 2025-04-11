package core

import (
    "bufio"
    "log"
    "net"
    "sync"

    "github.com/libp2p/go-reuseport"
    "github.com/reagin/double_ratchet/utils"
)

type Client struct {
    stopChan      chan bool
    waitGroup     sync.WaitGroup
    netConnect    net.Conn
    LocalAddress  string
    RemoteAddress string
    SendChannel   chan []byte
    RecvChannel   chan []byte
}

func NewClient(localAddress, remoteAddress string) *Client {
    return &Client{
        stopChan:      make(chan bool),
        LocalAddress:  localAddress,
        RemoteAddress: remoteAddress,
        SendChannel:   make(chan []byte, 8),
        RecvChannel:   make(chan []byte, 8),
    }
}

func (client *Client) StartClient() {
    go client.handleClient()
}

func (client *Client) StopClient() {
    log.Println("🛑 停止客户端中...")
    close(client.stopChan)

    if client.netConnect != nil {
        client.netConnect.Close()
    }
    client.waitGroup.Wait()

    log.Println("✅ 客户端完全退出")
}

func (client *Client) handleClient() {
    connect, err := reuseport.Dial("tcp", client.LocalAddress, client.RemoteAddress)
    if err != nil {
        log.Printf("❌ 客户端建立连接失败: %s\n", err.Error())
        return
    }
    defer connect.Close()
    client.netConnect = connect
    log.Printf("🎉 与服务端 %s 成功建立连接\n", client.RemoteAddress)

    // 记录双棘轮的状态信息
    ratchetState := utils.NewRatchetState()

    // 使用 bufio 修饰 net.Conn
    reader := bufio.NewReader(connect)
    writer := bufio.NewWriter(connect)

    // 生成 Diffe-Hellman 密钥对
    keyPair := utils.NewDiffeHellmanKeyPair()
    pubKeyBytes, _ := utils.EncodeMessage(keyPair.PublicKey.Bytes())

    // 向服务器发送初始公钥
    if _, err := writer.Write(pubKeyBytes); err != nil {
        log.Printf("🤯 向服务端发送信息失败: %s\n", err.Error())
        return
    }
    writer.Flush()

    // 接收服务端的初始公钥
    remotePubKeyBytes, _ := utils.DecodeMessage(reader)
    remotePubKey := utils.BytesToPublicKey(remotePubKeyBytes)

    // 计算共享密钥
    sharedSecret, _ := keyPair.PrivateKey.ECDH(remotePubKey)
    ratchetState.RootChain = sharedSecret

    // 生成 Diffe-Hellman 密钥对，用于后续主动发起请求
    keyPair = utils.NewDiffeHellmanKeyPair()
    pubKeyBytes, _ = utils.EncodeMessage(keyPair.PublicKey.Bytes())

    // 向服务器发送公钥
    if _, err := writer.Write(pubKeyBytes); err != nil {
        log.Printf("🤯 向服务端发送信息失败: %s\n", err.Error())
        return
    }
    writer.Flush()

    // 接收客户端的公钥
    remotePubKeyBytes, _ = utils.DecodeMessage(reader)
    remotePubKey = utils.BytesToPublicKey(remotePubKeyBytes)

    client.waitGroup.Add(2)
    // NOTE: 启动 gorunite 发送信息
    go startSendListener(client.stopChan, &client.waitGroup, client.SendChannel, writer, keyPair, ratchetState, &remotePubKey)
    // NOTE: 启动 gorunite 接收信息
    go startRecvListener(client.stopChan, &client.waitGroup, client.RecvChannel, reader, keyPair, ratchetState, &remotePubKey)

    <-client.stopChan
}
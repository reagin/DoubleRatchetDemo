package utils

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	"sync"
)

const (
	Sender = iota
	Receiver
)

type KeyChain struct {
	Count      int
	BaseKey    []byte
	MessageKey [][]byte
}

type RatchetMsg struct {
	Count     int
	Nonce     []byte
	Message   []byte
	PublicKey []byte
}

type RatchetState struct {
	Mutex       sync.Mutex  // 互斥锁
	RootChain   []byte      // 记录 RootChain
	SendChain   []*KeyChain // 记录 SendChain 的变化
	RecvChain   []*KeyChain // 记录 RecvChain 的变化
	SendCount   int         // 记录 SendChain 的迭代次数
	RecvCount   int         // 记录 RecvChain 的迭代次数
	RatchetType int         // 记录 RatchetState 的状态 (Sender|Receiver)
}

type DiffeHellmanKeyPair struct {
	PublicKey  *ecdh.PublicKey
	PrivateKey *ecdh.PrivateKey
}

func NewKeyChain() *KeyChain {
	return &KeyChain{
		Count:      -1, // 初始值设置为-1，便于后续获取对应的KeyChain
		BaseKey:    []byte{},
		MessageKey: [][]byte{},
	}
}

func NewRatchetMsg() *RatchetMsg {
	return &RatchetMsg{
		Count:     0,
		Nonce:     nil,
		Message:   nil,
		PublicKey: nil,
	}
}

func NewRatchetState() *RatchetState {
	return &RatchetState{
		Mutex:     sync.Mutex{},
		RootChain: nil,
		SendChain: []*KeyChain{},
		RecvChain: []*KeyChain{},
		SendCount: -1, //初始值设置为-1，便于判断 Sender||Receiver
		RecvCount: -1, // 初始值设置为-1，便于后续获取对应的KeyChain
	}
}

// 生成基于 Curve25519 的公私钥对
func NewDiffeHellmanKeyPair() *DiffeHellmanKeyPair {
	privKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
	pubKey := privKey.PublicKey()

	return &DiffeHellmanKeyPair{
		PublicKey:  pubKey,
		PrivateKey: privKey,
	}
}

func (dfk *DiffeHellmanKeyPair) UpdateKeyPair() {
	privKey, _ := ecdh.X25519().GenerateKey(rand.Reader)
	pubKey := privKey.PublicKey()

	dfk.PrivateKey = privKey
	dfk.PublicKey = pubKey
}

// 将 SendChain 中的 MessageKey 迭代 count 次
func (rs *RatchetState) StepSendMessageKey(times int) {
	keyChain := rs.SendChain[rs.SendCount]
	for range times {
		leftKey, rightKey := DevirateChainKey(keyChain.BaseKey, nil)
		keyChain.Count++
		keyChain.BaseKey = leftKey
		keyChain.MessageKey = append(keyChain.MessageKey, rightKey)
	}
}

// 将 RecvChain 中的 MessageKey 迭代 count 次
func (rs *RatchetState) StepRecvMessageKey(times int) {
	keyChain := rs.RecvChain[rs.RecvCount]
	for range times {
		leftKey, rightKey := DevirateChainKey(keyChain.BaseKey, nil)
		keyChain.Count++
		keyChain.BaseKey = leftKey
		keyChain.MessageKey = append(keyChain.MessageKey, rightKey)
	}
}

// 打印RatchetState状态
func (rs *RatchetState) DisplayRatchetState() {
	ratchetType := "Sender"
	if rs.RatchetType == Receiver {
		ratchetType = "Receiver"
	}
	fmt.Printf("Type: %s\n", ratchetType)
	fmt.Printf("RootChain: %x\n", rs.RootChain)
	fmt.Printf("SendChain Count: %v\n", rs.SendCount+1)
	for i := range rs.SendChain {
		keyChain := rs.SendChain[i]
		fmt.Printf("SendChain[%d] has %d items\n", i, keyChain.Count+1)
		for j := range keyChain.MessageKey {
			fmt.Printf("\tMessageChain[%d]: %x\n", j, keyChain.MessageKey[j])
		}
	}
	fmt.Printf("RecvChain Count: %v\n", rs.RecvCount+1)
	for i := range rs.RecvChain {
		keyChain := rs.RecvChain[i]
		fmt.Printf("RecvChain[%d] has %d items\n", i, keyChain.Count+1)
		for j := range keyChain.MessageKey {
			fmt.Printf("\tMessageChain[%d]: %x\n", j, keyChain.MessageKey[j])
		}
	}
	fmt.Println()
}

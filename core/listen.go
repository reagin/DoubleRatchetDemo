package core

import (
	"bufio"
	"crypto/ecdh"
	"encoding/json"
	"log"
	"sync"

	"github.com/reagin/double_ratchet/utils"
)

func startSendListener(isStop chan bool, wg *sync.WaitGroup, sendChannel chan []byte, writer *bufio.Writer, keyPair *utils.DiffeHellmanKeyPair, ratchetState *utils.RatchetState, remotePubKey **ecdh.PublicKey) {
	defer wg.Done()

	for {
		select {
		case <-isStop:
			log.Println("🛑 SendListener 接收监听器退出")
			return
		case message := <-sendChannel:
			// 增加互斥锁，避免竞争
			ratchetState.Mutex.Lock()
			// 根据初始棘轮状态判断当前是 Sender 还是 Receiver
			if ratchetState.SendCount < 0 && ratchetState.RecvCount < 0 {
				ratchetState.RatchetType = utils.Sender
			}

			switch ratchetState.RatchetType {
			case utils.Sender:
				// Sender: 如果SendCount等于RecvCount，则推进RootChain
				if ratchetState.SendCount == ratchetState.RecvCount {
					sharedSecret, _ := keyPair.PrivateKey.ECDH(*remotePubKey)
					leftKey, rightKey := utils.DevirateChainKey(ratchetState.RootChain, sharedSecret)
					// 迭代RootChain
					ratchetState.RootChain = leftKey
					// 迭代SendChain
					keyChain := utils.NewKeyChain()
					keyChain.BaseKey = rightKey
					ratchetState.SendCount++
					ratchetState.SendChain = append(ratchetState.SendChain, keyChain)
				}
				keyChain := ratchetState.SendChain[ratchetState.SendCount]
				// 迭代MessageKey
				ratchetState.StepSendMessageKey(1)
				// 加密信息
				nonce, ciphertext, err := utils.EncryptAESGCM(keyChain.MessageKey[keyChain.Count], message)
				if err != nil {
					log.Printf("❌ 加密信息失败: %x\n", message)
					return
				}
				// 组织棘轮信息结构
				ratchetMsg := utils.NewRatchetMsg()
				ratchetMsg.Count = keyChain.Count
				ratchetMsg.Nonce = nonce
				ratchetMsg.Message = ciphertext
				ratchetMsg.PublicKey = keyPair.PublicKey.Bytes()
				// 将待发送信息序列化，并发送信息
				message, _ = json.Marshal(ratchetMsg)
				message, _ = utils.EncodeMessage(message)
				if _, err := writer.Write(message); err != nil {
					log.Printf("🤯 发送信息失败: %s\n", err.Error())
					return
				}
				writer.Flush()
			case utils.Receiver:
				// Receiver: 如果SendCount不等于RecvCount，则更换密钥对，并推进RootChain
				if ratchetState.SendCount != ratchetState.RecvCount {
					// 更新DiffeHellman密钥对
					keyPair.UpdateKeyPair()

					sharedSecret, _ := keyPair.PrivateKey.ECDH(*remotePubKey)
					leftKey, rightKey := utils.DevirateChainKey(ratchetState.RootChain, sharedSecret)
					// 迭代RootChain
					ratchetState.RootChain = leftKey
					// 迭代SendChain
					keyChain := utils.NewKeyChain()
					keyChain.BaseKey = rightKey
					ratchetState.SendCount++
					ratchetState.SendChain = append(ratchetState.SendChain, keyChain)
				}
				keyChain := ratchetState.SendChain[ratchetState.SendCount]
				// 迭代MessageKey
				ratchetState.StepSendMessageKey(1)
				// 加密信息
				nonce, ciphertext, err := utils.EncryptAESGCM(keyChain.MessageKey[keyChain.Count], message)
				if err != nil {
					log.Printf("❌ 加密信息失败: %x\n", message)
					return
				}
				// 组织棘轮信息结构
				ratchetMsg := utils.NewRatchetMsg()
				ratchetMsg.Count = keyChain.Count
				ratchetMsg.Nonce = nonce
				ratchetMsg.Message = ciphertext
				ratchetMsg.PublicKey = keyPair.PublicKey.Bytes()
				// 将待发送信息序列化，并发送信息
				message, _ = json.Marshal(ratchetMsg)
				message, _ = utils.EncodeMessage(message)
				if _, err := writer.Write(message); err != nil {
					log.Printf("🤯 发送信息失败: %s\n", err.Error())
					return
				}
				writer.Flush()
			}

			// 解除互斥锁
			ratchetState.Mutex.Unlock()
		}
	}
}

func startRecvListener(isStop chan bool, wg *sync.WaitGroup, recvChannel chan []byte, reader *bufio.Reader, keyPair *utils.DiffeHellmanKeyPair, ratchetState *utils.RatchetState, remotePubKey **ecdh.PublicKey) {
	defer wg.Done()

	for {
		select {
		case <-isStop:
			log.Println("🛑 RecvListener 接收监听器退出")
			return
		default:
			// 定义棘轮信息结构
			ratchetMsg := utils.NewRatchetMsg()
			// 读取对方发送的字节流
			message, err := utils.DecodeMessage(reader)
			if err != nil {
				log.Printf("🤯 读取信息失败: %s\n", err.Error())
				return
			}
			// 解析对方发送的数据
			if err := json.Unmarshal(message, &ratchetMsg); err != nil {
				log.Printf("🤯 解析信息失败: %s\n", err.Error())
				return
			}
			*remotePubKey = utils.BytesToPublicKey(ratchetMsg.PublicKey)

			// 增加互斥锁，避免竞争
			ratchetState.Mutex.Lock()
			// 根据初始棘轮状态判断当前是 Sender 还是 Receiver
			if ratchetState.SendCount < 0 && ratchetState.RecvCount < 0 {
				ratchetState.RatchetType = utils.Receiver
			}

			switch ratchetState.RatchetType {
			case utils.Receiver:
				// Receiver: 如果SendCount等于RecvCount，则推进RootChain
				if ratchetState.SendCount == ratchetState.RecvCount {
					sharedSecret, _ := keyPair.PrivateKey.ECDH(*remotePubKey)
					leftKey, rightKey := utils.DevirateChainKey(ratchetState.RootChain, sharedSecret)
					// 迭代RootChain
					ratchetState.RootChain = leftKey
					// 迭代RecvChain
					keyChain := utils.NewKeyChain()
					keyChain.BaseKey = rightKey
					ratchetState.RecvCount++
					ratchetState.RecvChain = append(ratchetState.RecvChain, keyChain)
				}
				keyChain := ratchetState.RecvChain[ratchetState.RecvCount]
				// 迭代MessageKey
				ratchetState.StepRecvMessageKey(ratchetMsg.Count - keyChain.Count)
				// 解密信息
				plaintext, err := utils.DecryptAESGCM(keyChain.MessageKey[ratchetMsg.Count], ratchetMsg.Nonce, ratchetMsg.Message)
				if err != nil {
					log.Printf("❌ 解密信息失败: %s\n", err.Error())
					return
				}
				// 利用通道传输数据
				recvChannel <- plaintext
			case utils.Sender:
				// Sender: 如果SendCount不等于RecvCount，则推进RootChain，并更新密钥对
				if ratchetState.SendCount != ratchetState.RecvCount {
					sharedSecret, _ := keyPair.PrivateKey.ECDH(*remotePubKey)
					leftKey, rightKey := utils.DevirateChainKey(ratchetState.RootChain, sharedSecret)
					// 迭代RootChain
					ratchetState.RootChain = leftKey
					// 迭代RecvChain
					keyChain := utils.NewKeyChain()
					keyChain.BaseKey = rightKey
					ratchetState.RecvCount++
					ratchetState.RecvChain = append(ratchetState.RecvChain, keyChain)

					// 更新DiffeHellman密钥对
					keyPair.UpdateKeyPair()
				}
				keyChain := ratchetState.RecvChain[ratchetState.RecvCount]
				// 迭代MessageKey
				ratchetState.StepRecvMessageKey(ratchetMsg.Count - keyChain.Count)
				// 解密信息
				plaintext, err := utils.DecryptAESGCM(keyChain.MessageKey[ratchetMsg.Count], ratchetMsg.Nonce, ratchetMsg.Message)
				if err != nil {
					log.Printf("❌ 解密信息失败: %s\n", message)
					return
				}
				// 利用通道传输数据
				recvChannel <- plaintext
			}

			// 解除互斥锁
			ratchetState.Mutex.Unlock()
		}
	}
}

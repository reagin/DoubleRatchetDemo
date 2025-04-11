package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"io"
)

// 密钥派生函数，对于 RootChain 的派生应指定 salt 为 DH 计算的输出
func DevirateChainKey(key []byte, salt []byte) (leftKey []byte, rightKey []byte) {
	derivatedKey, err := hkdf.Key(sha256.New, key, salt, "", 64)
	if err != nil {
		return nil, nil
	}

	leftKey = derivatedKey[:32]
	rightKey = derivatedKey[32:]

	return leftKey, rightKey
}

// 使用密钥 key 加密明文信息
func EncryptAESGCM(key, plaintext []byte) (nonce []byte, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = aesGCM.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// 使用密钥 key 和 nonce 解密信息
func DecryptAESGCM(key, nonce, ciphertext []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err = aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// 将 []byte 格式的公钥转换为 *ecdh.PublicKey
func BytesToPublicKey(pubKey []byte) *ecdh.PublicKey {
	publicKey, err := ecdh.X25519().NewPublicKey(pubKey)
	if err != nil {
		return nil
	}
	return publicKey
}

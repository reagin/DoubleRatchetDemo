package utils

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
)

// Encode 编码消息
func EncodeMessage(message []byte) (msg []byte, err error) {
	// 读取消息的长度
	var length = uint32(len(message))
	var msgBuffer = new(bytes.Buffer)
	// 写入消息头部
	err = binary.Write(msgBuffer, binary.LittleEndian, length)
	if err != nil {
		return nil, err
	}
	// 写入消息实体
	err = binary.Write(msgBuffer, binary.LittleEndian, message)
	if err != nil {
		return nil, err
	}

	return msgBuffer.Bytes(), nil
}

// Decode 解码消息
func DecodeMessage(reader *bufio.Reader) (msg []byte, err error) {
	length := uint32(0)            // 读取消息的长度
	lengthBytes := make([]byte, 4) // 保存从reader中读取的数据

	// 读取前4个字节的长度数据
	if _, err = io.ReadFull(reader, lengthBytes); err != nil {
		return nil, err
	}

	// 读取消息头部
	err = binary.Read(bytes.NewBuffer(lengthBytes), binary.LittleEndian, &length)
	if err != nil {
		return nil, err
	}

	// 读取消息实体
	messageBytes := make([]byte, length)
	if _, err = io.ReadFull(reader, messageBytes); err != nil {
		return nil, err
	}

	return messageBytes, nil
}

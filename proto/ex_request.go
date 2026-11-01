package proto

import (
	"bytes"
	"encoding/binary"
)

// exReqHeader 是扩展市场协议专用的请求头布局，字段顺序与主站标准请求头不同，连续写入。
type exReqHeader struct {
	Head       uint8  // 报文首字节标识，固定取一。
	Customize  uint32 // 自定义四字节字段，固定填零。
	PacketType uint8  // 报文类型。
	PkgLen1    uint16 // 报文体长度冗余字段一，两字节。
	PkgLen2    uint16 // 报文体长度冗余字段二，取值与 PkgLen1 相同。
}

// buildExRequest 构造扩展市场协议请求报文：先把命令码与业务负载拼成报文体，
// 再写入记录报文体长度的专用请求头；编码失败时返回非空错误。
//
// 参数 `method` 为扩展市场协议命令码，`payload` 为业务负载字节流；返回拼装完成的
// 完整报文字节流，任一步二进制编码失败时返回错误。
func buildExRequest(method uint16, payload []byte) ([]byte, error) {
	// 命令码写入报文体，随后追加业务负载。
	body := new(bytes.Buffer)
	if err := binary.Write(body, binary.LittleEndian, method); err != nil {
		return nil, err
	}
	if _, err := body.Write(payload); err != nil {
		return nil, err
	}

	// 两个长度字段记录同一报文体长度，作为线上冗余校验。
	header := exReqHeader{
		Head:       0x01,
		Customize:  0,
		PacketType: 0x01,
		PkgLen1:    uint16(body.Len()),
		PkgLen2:    uint16(body.Len()),
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	_, err := buf.Write(body.Bytes())
	return buf.Bytes(), err
}

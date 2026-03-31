package proto

import (
	"bytes"
	"encoding/binary"
)

type exReqHeader struct {
	Head       uint8
	Customize  uint32
	PacketType uint8
	PkgLen1    uint16
	PkgLen2    uint16
}

func serializeExRequest(method uint16, payload []byte) ([]byte, error) {
	body := new(bytes.Buffer)
	if err := binary.Write(body, binary.LittleEndian, method); err != nil {
		return nil, err
	}
	if _, err := body.Write(payload); err != nil {
		return nil, err
	}

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

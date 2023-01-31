package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/bensema/gotdx/util"
)

type Hello2 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *Hello2Request
	reply      *Hello2Reply

	contentHex string
}

type Hello2Request struct {
}

type Hello2Reply struct {
	Info       string
	serverTime string
}

func NewHello2() *Hello2 {
	obj := new(Hello2)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(Hello2Request)
	obj.reply = new(Hello2Reply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_CMD2
	obj.contentHex = "d5d0c9ccd6a4a8af0000008fc22540130000d500c9ccbdf0d7ea00000002"
	return obj
}

func (obj *Hello2) Serialize() ([]byte, error) {
	b, err := hex.DecodeString(obj.contentHex)

	obj.reqHeader.PkgLen1 = 2 + uint16(len(b))
	obj.reqHeader.PkgLen2 = 2 + uint16(len(b))

	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	buf.Write(b)
	return buf.Bytes(), err
}

/*
0100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000011f85e34068747470733a2f2f626967352e6e65776f6e652e636f6d2e636e2f7a797968742f7a645f7a737a712e7a6970000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004150503a414c4c0d0a54494d453a303a30312d31353a30352c31353a30362d32333a35390d0a20202020c4facab9d3c3b5c4b0e6b1bebcb4bdabcda3d3c3a3acceaac1cbc4fab5c4d5fdb3a3cab9d3c32cc7ebbea1bfecc9fdd6c1d5d0c9ccd6a4c8af5043b0e6a1a30d0a20202020c8e7b9fbb2bbc4dcd7d4b6afc9fdbcb6a3acc7ebb5bdb9d9cdf868747470733a2f2f7777772e636d736368696e612e636f6d2fcfc2d4d8b0b2d7b0a3acd0bbd0bbc4fab5c4d6a7b3d6a3a100                                                                   年月日              年月日
*/
func (obj *Hello2) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	serverInfo := util.Utf8ToGbk(data[58:])
	//fmt.Println(hex.EncodeToString(data))
	obj.reply.Info = serverInfo
	return nil
}

func (obj *Hello2) Reply() *Hello2Reply {
	return obj.reply
}

package proto

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
)

type ExLogin struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExLoginReply
	contentHex string
}

type ExLoginReply struct {
	DateTime   string
	ServerName string
	Desc       string
	IP         string
}

func NewExLogin() *ExLogin {
	obj := &ExLogin{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExLoginReply),
		contentHex: "e5bb1c2fafe525941f32c6e5d53dfb415b734cc9cdbf0ac92021bfdd1eb06d22d008884c1611cb1378f6abd824d899d21f32c6e5d53dfb411f32c6e5d53dfb41a9325ac935dc0837335a16e4ce17c1bb",
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXLOGIN
	return obj
}

func (obj *ExLogin) BuildRequest() ([]byte, error) {
	payload, err := hex.DecodeString(obj.contentHex)
	if err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXLOGIN, payload)
}

func (obj *ExLogin) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 294 {
		return fmt.Errorf("invalid ex login response length: %d", len(data))
	}

	year := binary.LittleEndian.Uint16(data[53:55])
	month := int(data[55])
	day := int(data[56])
	minute := int(data[57])
	hour := int(data[58])
	second := int(data[60])

	obj.reply.DateTime = time.Date(int(year), time.Month(month), day, hour, minute, second, 0, time.Local).Format("2006-01-02 15:04:05")
	obj.reply.ServerName = Utf8ToGbk(data[61:82])
	obj.reply.Desc = Utf8ToGbk(data[93:244])
	obj.reply.IP = Utf8ToGbk(data[242:])
	return nil
}

func (obj *ExLogin) Response() *ExLoginReply {
	return obj.reply
}

type ExServerInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExServerInfoReply
}

type ExServerInfoReply struct {
	Delay      uint32
	Info       string
	Version    string
	ServerSign string
	TimeNow    string
	ServerName string
}

func NewExServerInfo() *ExServerInfo {
	obj := &ExServerInfo{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExServerInfoReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXSERVERINFO
	return obj
}

func (obj *ExServerInfo) BuildRequest() ([]byte, error) {
	return buildExRequest(KMSG_EXSERVERINFO, nil)
}

func (obj *ExServerInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 327 {
		return fmt.Errorf("invalid ex server info response length: %d", len(data))
	}

	obj.reply.Delay = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Info = Utf8ToGbk(data[16:41])
	obj.reply.Version = Utf8ToGbk(data[41:70])
	obj.reply.ServerSign = Utf8ToGbk(data[117:130])

	dateNow := binary.LittleEndian.Uint32(data[80:84])
	timeNow := binary.LittleEndian.Uint32(data[84:88])
	year := int(dateNow / 10000)
	month := int((dateNow % 10000) / 100)
	day := int(dateNow % 100)
	hour := int(timeNow / 10000)
	minute := int((timeNow % 10000) / 100)
	second := int(timeNow % 100)
	obj.reply.TimeNow = time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local).Format("2006-01-02 15:04:05")
	obj.reply.ServerName = Utf8ToGbk(data[159:189])
	return nil
}

func (obj *ExServerInfo) Response() *ExServerInfoReply {
	return obj.reply
}

package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

type ExchangeAnnouncement struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *ExchangeAnnouncementReply
}

type ExchangeAnnouncementReply struct {
	Version uint8
	Content string
}

func NewExchangeAnnouncement() *ExchangeAnnouncement {
	obj := &ExchangeAnnouncement{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(ExchangeAnnouncementReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXCHANGEANNOUNCE
	return obj
}

func (obj *ExchangeAnnouncement) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2
	obj.reqHeader.PkgLen2 = 2
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	return buf.Bytes(), err
}

func (obj *ExchangeAnnouncement) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) == 0 {
		return fmt.Errorf("invalid exchange announcement response length: %d", len(data))
	}
	obj.reply.Version = data[0]
	obj.reply.Content = Utf8ToGbk(data[1:])
	return nil
}

func (obj *ExchangeAnnouncement) Reply() *ExchangeAnnouncementReply {
	return obj.reply
}

type HeartBeat struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *HeartBeatReply
}

type HeartBeatReply struct {
	Date uint32
}

func NewHeartBeat() *HeartBeat {
	obj := &HeartBeat{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(HeartBeatReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_HEARTBEAT
	return obj
}

func (obj *HeartBeat) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2
	obj.reqHeader.PkgLen2 = 2
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	return buf.Bytes(), err
}

func (obj *HeartBeat) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 10 {
		return fmt.Errorf("invalid heartbeat response length: %d", len(data))
	}
	obj.reply.Date = binary.LittleEndian.Uint32(data[6:10])
	return nil
}

func (obj *HeartBeat) Reply() *HeartBeatReply {
	return obj.reply
}

type Announcement struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *AnnouncementReply
}

type AnnouncementReply struct {
	HasContent bool
	ExpireDate string
	Title      string
	Author     string
	Content    string
}

func NewAnnouncement() *Announcement {
	obj := &Announcement{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(AnnouncementReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_ANNOUNCEMENT
	return obj
}

func (obj *Announcement) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 56
	obj.reqHeader.PkgLen2 = 56
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	_, err := buf.Write(make([]byte, 54))
	return buf.Bytes(), err
}

func (obj *Announcement) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) == 0 {
		return fmt.Errorf("invalid announcement response length: %d", len(data))
	}
	if data[0] != 0x01 {
		return nil
	}
	if len(data) < 11 {
		return fmt.Errorf("invalid announcement payload length: %d", len(data))
	}
	dateRaw := binary.LittleEndian.Uint32(data[1:5])
	titleLen := int(binary.LittleEndian.Uint16(data[5:7]))
	authorLen := int(binary.LittleEndian.Uint16(data[7:9]))
	contentLen := int(binary.LittleEndian.Uint16(data[9:11]))
	end := 11 + titleLen + authorLen + contentLen
	if end > len(data) {
		return fmt.Errorf("invalid announcement text length: %d", len(data))
	}
	obj.reply.HasContent = true
	obj.reply.ExpireDate = fmt.Sprintf("%04d-%02d-%02d", dateRaw/10000, (dateRaw%10000)/100, dateRaw%100)
	pos := 11
	obj.reply.Title = Utf8ToGbk(data[pos : pos+titleLen])
	pos += titleLen
	obj.reply.Author = Utf8ToGbk(data[pos : pos+authorLen])
	pos += authorLen
	obj.reply.Content = Utf8ToGbk(data[pos : pos+contentLen])
	return nil
}

func (obj *Announcement) Reply() *AnnouncementReply {
	return obj.reply
}

type Info struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *InfoReply
}

type InfoReply struct {
	Delay       uint32
	Info        string
	Content     string
	ServerSign  string
	TimeNow     string
	Region      uint16
	MaybeSwitch uint16
}

func NewInfo() *Info {
	obj := &Info{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(InfoReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_PING
	return obj
}

func (obj *Info) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2
	obj.reqHeader.PkgLen2 = 2
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	return buf.Bytes(), err
}

func (obj *Info) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 427 {
		return fmt.Errorf("invalid info response length: %d", len(data))
	}

	obj.reply.Delay = binary.LittleEndian.Uint32(data[0:4])
	obj.reply.Info = Utf8ToGbk(data[16:71])
	obj.reply.Content = Utf8ToGbk(data[81:336])
	obj.reply.ServerSign = Utf8ToGbk(data[336:356])
	dateNow := binary.LittleEndian.Uint32(data[397:401])
	timeNow := binary.LittleEndian.Uint32(data[401:405])
	if dateNow != 0 {
		ts := time.Date(
			int(dateNow/10000),
			time.Month((dateNow%10000)/100),
			int(dateNow%100),
			int(timeNow/10000),
			int((timeNow%10000)/100),
			int(timeNow%100),
			0,
			time.Local,
		)
		obj.reply.TimeNow = ts.Format("2006-01-02 15:04:05")
	}
	obj.reply.Region = binary.LittleEndian.Uint16(data[389:391])
	obj.reply.MaybeSwitch = binary.LittleEndian.Uint16(data[395:397])
	return nil
}

func (obj *Info) Reply() *InfoReply {
	return obj.reply
}

package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
)

type RawDataReply struct {
	Length int
	Data   []byte
	Hex    string
}

type RawMainProtocol struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	reply      *RawDataReply
	method     uint16
	payload    []byte
	generic    bool
}

func newRawMainProtocol(method uint16, payload []byte) *RawMainProtocol {
	obj := &RawMainProtocol{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		reply:      new(RawDataReply),
		method:     method,
		payload:    append([]byte(nil), payload...),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = method
	return obj
}

func newGenericRawMainProtocol(method uint16, payload []byte) *RawMainProtocol {
	obj := newRawMainProtocol(method, payload)
	obj.generic = true
	return obj
}

func (obj *RawMainProtocol) Serialize() ([]byte, error) {
	if obj.generic {
		return serializeGenericRequest(0x0c, 0, 0x01, obj.method, obj.payload)
	}
	obj.reqHeader.PkgLen1 = uint16(2 + len(obj.payload))
	obj.reqHeader.PkgLen2 = obj.reqHeader.PkgLen1
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	_, err := buf.Write(obj.payload)
	return buf.Bytes(), err
}

func (obj *RawMainProtocol) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	obj.reply.Length = len(data)
	obj.reply.Data = append([]byte(nil), data...)
	obj.reply.Hex = hex.EncodeToString(data)
	return nil
}

func (obj *RawMainProtocol) Reply() *RawDataReply {
	return obj.reply
}

func NewTodoB() *RawMainProtocol {
	return newRawMainProtocol(KMSG_TODOB, mustDecodeHex(
		"e53878ee8bd8dbb8"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"901a1266bd9f62d9"+
			"6810db2bdf3e50a1"+
			"9e93269128ddf91f"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"27fd50435e32ca0d"+
			"8872a27c327343f1"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"749933ae27700357"+
			"7c8810a76fd73daf",
	))
}

func NewTodoFDE() *RawMainProtocol { return newRawMainProtocol(KMSG_TODOFDE, nil) }
func NewClient264B() *RawMainProtocol {
	return newGenericRawMainProtocol(KMSG_CLIENT264B, []byte{0x00, 0x64})
}
func NewClient26AE() *RawMainProtocol {
	return newGenericRawMainProtocol(KMSG_CLIENT26AE, []byte{0x00, 0x64})
}
func NewClient26B1() *RawMainProtocol {
	return newGenericRawMainProtocol(KMSG_CLIENT26B1, []byte{0x00, 0x64, 0x00})
}

func NewClient26AC() *RawMainProtocol {
	payload := new(bytes.Buffer)
	_ = binary.Write(payload, binary.LittleEndian, uint8(0))
	_ = binary.Write(payload, binary.LittleEndian, uint8(100))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1234))
	payload.Write(make([]byte, 24))
	payload.Write([]byte{192, 168, 0, 1})
	mac := [6]byte{'m', 'a', 'c', 'a', 'd', 'd'}
	_ = binary.Write(payload, binary.LittleEndian, mac)
	_ = binary.Write(payload, binary.LittleEndian, uint16(40))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1864))
	_ = binary.Write(payload, binary.LittleEndian, uint16(0))
	return newGenericRawMainProtocol(KMSG_CLIENT26AC, payload.Bytes())
}

func NewClient26AD() *RawMainProtocol {
	payload := new(bytes.Buffer)
	_ = binary.Write(payload, binary.LittleEndian, uint8(0))
	_ = binary.Write(payload, binary.LittleEndian, uint8(100))
	tag := [16]byte{'T', 'D', 'X', 'W'}
	_ = binary.Write(payload, binary.LittleEndian, tag)
	payload.Write(make([]byte, 24))
	payload.Write([]byte{192, 168, 0, 1})
	mac := [6]byte{'m', 'a', 'c', 'a', 'd', 'd'}
	_ = binary.Write(payload, binary.LittleEndian, mac)
	tag2 := [16]byte{'T', 'D', 'X', 'W'}
	_ = binary.Write(payload, binary.LittleEndian, tag2)
	_ = binary.Write(payload, binary.LittleEndian, uint16(40))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1864))
	_ = binary.Write(payload, binary.LittleEndian, uint16(0))
	_ = binary.Write(payload, binary.LittleEndian, uint16(1234))
	_ = binary.Write(payload, binary.LittleEndian, uint8(1))
	token := [17]byte{'O', 'P', 'r', '2', 'I', 'H', 'Z', '5', 'r', '3', 'l', 'u', 'K', '9', 'K', 'a'}
	_ = binary.Write(payload, binary.LittleEndian, token)
	_ = binary.Write(payload, binary.LittleEndian, uint8(2))
	_ = binary.Write(payload, binary.LittleEndian, uint8(1))
	return newGenericRawMainProtocol(KMSG_CLIENT26AD, payload.Bytes())
}

type GetSecurityFeature452 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityFeature452Request
	reply      *GetSecurityFeature452Reply
}

type GetSecurityFeature452Request struct {
	Start uint32
	Count uint32
	One   uint32
	Zero  uint16
}

type GetSecurityFeature452Reply struct {
	Count uint16
	List  []SecurityFeature452Item
}

type SecurityFeature452Item struct {
	Market uint8
	Code   string
	P1     float64
	P2     float64
}

func NewGetSecurityFeature452() *GetSecurityFeature452 {
	obj := &GetSecurityFeature452{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &GetSecurityFeature452Request{Count: 2000, One: 1},
		reply:      new(GetSecurityFeature452Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYFEATURE452
	return obj
}

func (obj *GetSecurityFeature452) SetParams(req *GetSecurityFeature452Request) {
	if req.Count == 0 {
		req.Count = 2000
	}
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *GetSecurityFeature452) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 16
	obj.reqHeader.PkgLen2 = 16
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	err := binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetSecurityFeature452) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 2 {
		return fmt.Errorf("invalid 452 response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+13 > len(data) {
			return fmt.Errorf("invalid 452 item %d", i)
		}
		codeNum := binary.LittleEndian.Uint32(data[pos+1 : pos+5])
		item := SecurityFeature452Item{
			Market: data[pos],
			Code:   fmt.Sprintf("%d", codeNum),
			P1:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+5 : pos+9]))),
			P2:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+9 : pos+13]))),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 13
	}
	return nil
}

func (obj *GetSecurityFeature452) Reply() *GetSecurityFeature452Reply { return obj.reply }

type ExGetListExtra struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetListExtraRequest
	reply      *ExGetListExtraReply
}

type ExGetListExtraRequest struct {
	A     uint16
	B     uint16
	Count uint16
}

type ExGetListExtraReply struct {
	Start uint32
	Count uint16
	List  []ExExtraListItem
}

type ExExtraListItem struct {
	Category uint8
	Code     string
	Flag     uint8
	Values   []uint16
}

func NewExGetListExtra() *ExGetListExtra {
	obj := &ExGetListExtra{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &ExGetListExtraRequest{Count: 500},
		reply:      new(ExGetListExtraReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXLIST_EXTRA
	return obj
}

func (obj *ExGetListExtra) SetParams(req *ExGetListExtraRequest) {
	if req.Count == 0 {
		req.Count = 500
	}
	obj.request = req
}

func (obj *ExGetListExtra) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXLIST_EXTRA, payload.Bytes())
}

func (obj *ExGetListExtra) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 6 {
		return fmt.Errorf("invalid 23f6 response length: %d", len(data))
	}
	obj.reply.Start = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Count = binary.LittleEndian.Uint16(data[4:6])
	pos := 6
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+34 > len(data) {
			return fmt.Errorf("invalid 23f6 item %d", i)
		}
		item := ExExtraListItem{
			Category: data[pos],
			Code:     Utf8ToGbk(data[pos+1 : pos+9]),
			Flag:     data[pos+9],
		}
		item.Values = make([]uint16, 12)
		for j := 0; j < 12; j++ {
			item.Values[j] = binary.LittleEndian.Uint16(data[pos+10+j*2 : pos+12+j*2])
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 34
	}
	return nil
}

func (obj *ExGetListExtra) Reply() *ExGetListExtraReply { return obj.reply }

type ExExperiment2487 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExExperiment2487Request
	reply      *ExExperiment2487Reply
}

type ExExperiment2487Request struct {
	Category uint8
	Code     [23]byte
}

type ExExperiment2487Reply struct {
	Category uint8
	Code     string
	Active   uint32
	PreClose float64
	Open     float64
	High     float64
	Low      float64
	Close    float64
	U1       float64
	Price    float64
	Vol      uint32
	CurVol   uint32
	Amount   float64
	TailHex  string
}

func NewExExperiment2487() *ExExperiment2487 {
	obj := &ExExperiment2487{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExExperiment2487Request),
		reply:      new(ExExperiment2487Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES_EXPERIMENT1
	return obj
}

func (obj *ExExperiment2487) SetParams(req *ExExperiment2487Request) { obj.request = req }

func (obj *ExExperiment2487) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXQUOTES_EXPERIMENT1, payload.Bytes())
}

func (obj *ExExperiment2487) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 68 {
		return fmt.Errorf("invalid 2487 response length: %d", len(data))
	}
	obj.reply.Category = data[0]
	obj.reply.Code = Utf8ToGbk(data[1:24])
	obj.reply.Active = binary.LittleEndian.Uint32(data[24:28])
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[28:32])))
	obj.reply.Open = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[32:36])))
	obj.reply.High = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[36:40])))
	obj.reply.Low = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[40:44])))
	obj.reply.Close = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[44:48])))
	obj.reply.U1 = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[48:52])))
	obj.reply.Price = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[52:56])))
	obj.reply.Vol = binary.LittleEndian.Uint32(data[56:60])
	obj.reply.CurVol = binary.LittleEndian.Uint32(data[60:64])
	obj.reply.Amount = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[64:68])))
	if len(data) > 68 {
		obj.reply.TailHex = hex.EncodeToString(data[68:])
	}
	return nil
}

func (obj *ExExperiment2487) Reply() *ExExperiment2487Reply { return obj.reply }

type ExExperiment2488 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExExperiment2488Request
	reply      *ExExperiment2488Reply
}

type ExExperiment2488Request struct {
	Category uint8
	Code     [23]byte
	Zero1    uint32
	Mode     uint16
	Zero2    uint32
	Zero3    uint32
}

type ExExperiment2488Reply struct {
	Category uint8
	Code     string
	Count    uint16
	List     []ExExperiment2488Item
}

type ExExperiment2488Item struct {
	ID     uint32
	Values []uint16
}

func NewExExperiment2488() *ExExperiment2488 {
	obj := &ExExperiment2488{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &ExExperiment2488Request{Mode: 55},
		reply:      new(ExExperiment2488Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXQUOTES_EXPERIMENT2
	return obj
}

func (obj *ExExperiment2488) SetParams(req *ExExperiment2488Request) {
	if req.Mode == 0 {
		req.Mode = 55
	}
	obj.request = req
}

func (obj *ExExperiment2488) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXQUOTES_EXPERIMENT2, payload.Bytes())
}

func (obj *ExExperiment2488) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 38 {
		return fmt.Errorf("invalid 2488 response length: %d", len(data))
	}
	obj.reply.Category = data[0]
	obj.reply.Code = Utf8ToGbk(data[1:36])
	obj.reply.Count = binary.LittleEndian.Uint16(data[36:38])
	pos := 38
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+16 > len(data) {
			return fmt.Errorf("invalid 2488 item %d", i)
		}
		item := ExExperiment2488Item{
			ID:     binary.LittleEndian.Uint32(data[pos : pos+4]),
			Values: make([]uint16, 6),
		}
		for j := 0; j < 6; j++ {
			item.Values[j] = binary.LittleEndian.Uint16(data[pos+4+j*2 : pos+6+j*2])
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 16
	}
	return nil
}

func (obj *ExExperiment2488) Reply() *ExExperiment2488Reply { return obj.reply }

type ExGetKLine2 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetKLine2Request
	reply      *ExGetKLine2Reply
}

type ExGetKLine2Request struct {
	Category uint8
	Code     [23]byte
	Period   uint16
	Times    uint16
	Start    uint32
	Count    uint32
	Reserved [16]byte
}

type ExGetKLine2Reply struct {
	Category uint8
	Name     string
	Period   uint16
	Times    uint16
	Start    uint32
	Count    uint16
	List     []ExKLineItem
}

func NewExGetKLine2() *ExGetKLine2 {
	obj := &ExGetKLine2{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetKLine2Request),
		reply:      new(ExGetKLine2Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXKLINE2
	return obj
}

func (obj *ExGetKLine2) SetParams(req *ExGetKLine2Request) {
	if req.Times == 0 {
		req.Times = 1
	}
	if req.Count == 0 {
		req.Count = 800
	}
	obj.request = req
}

func (obj *ExGetKLine2) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXKLINE2, payload.Bytes())
}

func (obj *ExGetKLine2) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 42 {
		return fmt.Errorf("invalid ex kline2 response length: %d", len(data))
	}
	obj.reply.Category = data[0]
	obj.reply.Name = Utf8ToGbk(data[1:24])
	obj.reply.Period = binary.LittleEndian.Uint16(data[24:26])
	obj.reply.Times = binary.LittleEndian.Uint16(data[26:28])
	obj.reply.Start = binary.LittleEndian.Uint32(data[28:32])
	obj.reply.Count = binary.LittleEndian.Uint16(data[40:42])
	pos := 42
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+32 > len(data) {
			return fmt.Errorf("invalid ex kline2 item %d", i)
		}
		dateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
		ts, ok := decodeDateNum(obj.reply.Period, dateNum)
		if !ok {
			return fmt.Errorf("invalid ex kline2 datetime: %d", dateNum)
		}
		item := ExKLineItem{
			DateTime: ts.Format("2006-01-02 15:04:05"),
			Open:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))),
			High:     float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))),
			Low:      float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))),
			Close:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+16 : pos+20]))),
			Amount:   float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+20 : pos+24]))),
			Vol:      binary.LittleEndian.Uint32(data[pos+24 : pos+28]),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 32
	}
	return nil
}

func (obj *ExGetKLine2) Reply() *ExGetKLine2Reply { return obj.reply }

type ExMapping2562 struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExMapping2562Request
	reply      *ExMapping2562Reply
}

type ExMapping2562Request struct {
	Market uint16
	Start  uint32
	Count  uint32
}

type ExMapping2562Reply struct {
	Count uint16
	List  []ExMapping2562Item
}

type ExMapping2562Item struct {
	Category uint16
	Name     string
	Unknown  uint16
	Index    uint32
	Switch   uint8
	Code1    float64
	Code2    float64
	Code3    float64
	Code4    uint16
	Code5    uint16
}

func NewExMapping2562() *ExMapping2562 {
	obj := &ExMapping2562{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    &ExMapping2562Request{Count: 600},
		reply:      new(ExMapping2562Reply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXMAPPING2562
	return obj
}

func (obj *ExMapping2562) SetParams(req *ExMapping2562Request) {
	if req.Count == 0 {
		req.Count = 600
	}
	obj.request = req
}

func (obj *ExMapping2562) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXMAPPING2562, payload.Bytes())
}

func (obj *ExMapping2562) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 2 {
		return fmt.Errorf("invalid 2562 response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+48 > len(data) {
			return fmt.Errorf("invalid 2562 item %d", i)
		}
		item := ExMapping2562Item{
			Category: binary.LittleEndian.Uint16(data[pos : pos+2]),
			Name:     Utf8ToGbk(data[pos+2 : pos+25]),
			Unknown:  binary.LittleEndian.Uint16(data[pos+25 : pos+27]),
			Index:    binary.LittleEndian.Uint32(data[pos+27 : pos+31]),
			Switch:   data[pos+31],
			Code1:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+32 : pos+36]))),
			Code2:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+36 : pos+40]))),
			Code3:    float64(math.Float32frombits(binary.LittleEndian.Uint32(data[pos+40 : pos+44]))),
			Code4:    binary.LittleEndian.Uint16(data[pos+44 : pos+46]),
			Code5:    binary.LittleEndian.Uint16(data[pos+46 : pos+48]),
		}
		obj.reply.List = append(obj.reply.List, item)
		pos += 48
	}
	return nil
}

func (obj *ExMapping2562) Reply() *ExMapping2562Reply { return obj.reply }

func mustDecodeHex(value string) []byte {
	out, err := hex.DecodeString(value)
	if err != nil {
		panic(err)
	}
	return out
}

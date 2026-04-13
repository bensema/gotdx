package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type GetQuotesEncrypt struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetQuotesEncryptRequest
	reply      *GetQuotesEncryptReply
}

type GetQuotesEncryptRequest struct {
	Stocks []Stock
}

type GetQuotesEncryptReply struct {
	Count uint16
	List  []EncryptedQuoteItem
}

type EncryptedQuoteItem struct {
	Market     uint8
	Code       string
	Active     uint16
	Close      float64
	PreClose   float64
	Open       float64
	High       float64
	Low        float64
	Time       string
	Vol        int
	CurVol     int
	Amount     float64
	InVol      int
	OutVol     int
	SAmount    int
	OpenAmount int
	BidLevels  []Level
	AskLevels  []Level
}

func NewGetQuotesEncrypt(req *GetQuotesEncryptRequest) *GetQuotesEncrypt {
	obj := &GetQuotesEncrypt{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetQuotesEncryptRequest),
		reply:      new(GetQuotesEncryptReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_QUOTESENCRYPT
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetQuotesEncrypt) applyRequest(req *GetQuotesEncryptRequest) {
	obj.request = req
}

func (obj *GetQuotesEncrypt) BuildRequest() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(obj.request.Stocks))); err != nil {
		return nil, err
	}
	for _, stock := range obj.request.Stocks {
		code := make([]byte, 6)
		copy(code, stock.Code)
		if err := binary.Write(buf, binary.LittleEndian, stock.Market); err != nil {
			return nil, err
		}
		if _, err := buf.Write(code); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(22234)); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
			return nil, err
		}
	}
	return buildGenericRequest(0x0c, 0, 0x01, KMSG_QUOTESENCRYPT, buf.Bytes())
}

func (obj *GetQuotesEncrypt) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	xor := make([]byte, len(data))
	for i := range data {
		xor[i] = data[i] ^ 0x93
	}
	if len(xor) < 2 {
		return fmt.Errorf("invalid encrypted quotes response length: %d", len(xor))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(xor[:2])
	pos := 2
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+9 > len(xor) {
			return fmt.Errorf("invalid encrypted quote item %d", i)
		}
		item := EncryptedQuoteItem{
			Market: xor[pos],
			Code:   Utf8ToGbk(xor[pos+1 : pos+7]),
			Active: binary.LittleEndian.Uint16(xor[pos+7 : pos+9]),
		}
		pos += 9
		closeRaw := getprice(xor, &pos)
		preCloseDiff := getprice(xor, &pos)
		openDiff := getprice(xor, &pos)
		highDiff := getprice(xor, &pos)
		lowDiff := getprice(xor, &pos)
		if pos+4 > len(xor) {
			return fmt.Errorf("invalid encrypted quote time %d", i)
		}
		timeRaw := binary.LittleEndian.Uint32(xor[pos : pos+4])
		pos += 4
		_ = getprice(xor, &pos)
		item.Vol = getprice(xor, &pos)
		item.CurVol = getprice(xor, &pos)
		item.Amount = getfloat32(xor, &pos)
		item.InVol = getprice(xor, &pos)
		item.OutVol = getprice(xor, &pos)
		item.SAmount = getprice(xor, &pos)
		item.OpenAmount = getprice(xor, &pos)
		for j := 0; j < 5; j++ {
			bid := getprice(xor, &pos) + closeRaw
			ask := getprice(xor, &pos) + closeRaw
			bidVol := getprice(xor, &pos)
			askVol := getprice(xor, &pos)
			item.BidLevels = append(item.BidLevels, Level{Price: float64(bid) / 100.0, Vol: bidVol})
			item.AskLevels = append(item.AskLevels, Level{Price: float64(ask) / 100.0, Vol: askVol})
		}
		if pos+10 > len(xor) {
			return fmt.Errorf("invalid encrypted quote tail %d", i)
		}
		pos += 10
		for j := 0; j < 6; j++ {
			_ = getprice(xor, &pos)
			_ = getprice(xor, &pos)
			_ = getprice(xor, &pos)
			_ = getprice(xor, &pos)
		}

		item.Close = float64(closeRaw) / 100.0
		item.PreClose = float64(closeRaw+preCloseDiff) / 100.0
		item.Open = float64(closeRaw+openDiff) / 100.0
		item.High = float64(closeRaw+highDiff) / 100.0
		item.Low = float64(closeRaw+lowDiff) / 100.0
		item.Time = fmt.Sprintf("%02d:%02d:%02d", timeRaw/10000, (timeRaw/100)%100, timeRaw%100)
		obj.reply.List = append(obj.reply.List, item)
	}
	return nil
}

func (obj *GetQuotesEncrypt) Response() *GetQuotesEncryptReply {
	return obj.reply
}

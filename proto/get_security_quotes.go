package proto

import (
	"bytes"
	"encoding/binary"
)

type GetSecurityQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityQuotesRequest
	reply      *GetSecurityQuotesReply
}

type Stock struct {
	Market uint8
	Code   string
}

type GetSecurityQuotesRequest struct {
	StockList []Stock
}

type GetSecurityQuotesReply struct {
	Count uint16
	List  []SecurityQuote
}

type SecurityQuote struct {
	Market     uint8
	Code       string
	Active1    uint16
	Close      float64
	Price      float64
	PreClose   float64
	LastClose  float64
	Open       float64
	High       float64
	Low        float64
	ServerTime string
	NegPrice   float64
	Vol        int
	CurVol     int
	Amount     float64
	SVol       int
	BVol       int
	SAmount    int
	OpenAmount int
	BidLevels  []Level
	AskLevels  []Level
	Bid1       float64
	Ask1       float64
	BidVol1    int
	AskVol1    int
	Bid2       float64
	Ask2       float64
	BidVol2    int
	AskVol2    int
	Bid3       float64
	Ask3       float64
	BidVol3    int
	AskVol3    int
	Bid4       float64
	Ask4       float64
	BidVol4    int
	AskVol4    int
	Bid5       float64
	Ask5       float64
	BidVol5    int
	AskVol5    int
	Unknown    int16
	Rate       float64
	Active2    uint16
}

type Level struct {
	Price float64
	Vol   int
}

func NewGetSecurityQuotes() *GetSecurityQuotes {
	obj := new(GetSecurityQuotes)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityQuotesRequest)
	obj.reply = new(GetSecurityQuotesReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_SECURITYQUOTES
	return obj
}

func (obj *GetSecurityQuotes) SetParams(req *GetSecurityQuotesRequest) {
	obj.request = req
}

func (obj *GetSecurityQuotes) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 2 + uint16(len(obj.request.StockList)*7) + 10
	obj.reqHeader.PkgLen2 = 2 + uint16(len(obj.request.StockList)*7) + 10

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	if err != nil {
		return nil, err
	}

	if err := binary.Write(buf, binary.LittleEndian, uint16(5)); err != nil {
		return nil, err
	}
	if _, err := buf.Write(make([]byte, 6)); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(len(obj.request.StockList))); err != nil {
		return nil, err
	}

	for _, stock := range obj.request.StockList {
		code := make([]byte, 6)
		copy(code, stock.Code)
		if err := binary.Write(buf, binary.LittleEndian, stock.Market); err != nil {
			return nil, err
		}
		if _, err := buf.Write(code); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (obj *GetSecurityQuotes) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	pos := 0
	pos += 2
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2

	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := SecurityQuote{}
		if err := binary.Read(bytes.NewBuffer(data[pos:pos+1]), binary.LittleEndian, &ele.Market); err != nil {
			return err
		}
		pos++

		var code [6]byte
		if err := binary.Read(bytes.NewBuffer(data[pos:pos+6]), binary.LittleEndian, &code); err != nil {
			return err
		}
		ele.Code = Utf8ToGbk(code[:])
		pos += 6

		if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Active1); err != nil {
			return err
		}
		pos += 2

		basePrice := getprice(data, &pos)
		preCloseDiff := getprice(data, &pos)
		openDiff := getprice(data, &pos)
		highDiff := getprice(data, &pos)
		lowDiff := getprice(data, &pos)
		serverTimeRaw := getprice(data, &pos)
		negPriceRaw := getprice(data, &pos)
		vol := getprice(data, &pos)
		curVol := getprice(data, &pos)
		amount := getfloat32(data, &pos)
		sVol := getprice(data, &pos)
		bVol := getprice(data, &pos)
		sAmount := getprice(data, &pos)
		openAmount := getprice(data, &pos)

		ele.Close = float64(basePrice) / 100.0
		ele.Price = ele.Close
		ele.PreClose = float64(basePrice+preCloseDiff) / 100.0
		ele.LastClose = ele.PreClose
		ele.Open = float64(basePrice+openDiff) / 100.0
		ele.High = float64(basePrice+highDiff) / 100.0
		ele.Low = float64(basePrice+lowDiff) / 100.0
		ele.ServerTime = formatServerTime(serverTimeRaw)
		ele.NegPrice = float64(negPriceRaw) / 100.0
		ele.Vol = vol
		ele.CurVol = curVol
		ele.Amount = amount
		ele.SVol = sVol
		ele.BVol = bVol
		ele.SAmount = sAmount
		ele.OpenAmount = openAmount

		for i := 0; i < 5; i++ {
			bid := getprice(data, &pos) + basePrice
			ask := getprice(data, &pos) + basePrice
			bidVol := getprice(data, &pos)
			askVol := getprice(data, &pos)

			ele.BidLevels = append(ele.BidLevels, Level{
				Price: float64(bid) / 100.0,
				Vol:   bidVol,
			})
			ele.AskLevels = append(ele.AskLevels, Level{
				Price: float64(ask) / 100.0,
				Vol:   askVol,
			})
		}

		ele.Bid1 = ele.BidLevels[0].Price
		ele.Bid2 = ele.BidLevels[1].Price
		ele.Bid3 = ele.BidLevels[2].Price
		ele.Bid4 = ele.BidLevels[3].Price
		ele.Bid5 = ele.BidLevels[4].Price
		ele.Ask1 = ele.AskLevels[0].Price
		ele.Ask2 = ele.AskLevels[1].Price
		ele.Ask3 = ele.AskLevels[2].Price
		ele.Ask4 = ele.AskLevels[3].Price
		ele.Ask5 = ele.AskLevels[4].Price
		ele.BidVol1 = ele.BidLevels[0].Vol
		ele.BidVol2 = ele.BidLevels[1].Vol
		ele.BidVol3 = ele.BidLevels[2].Vol
		ele.BidVol4 = ele.BidLevels[3].Vol
		ele.BidVol5 = ele.BidLevels[4].Vol
		ele.AskVol1 = ele.AskLevels[0].Vol
		ele.AskVol2 = ele.AskLevels[1].Vol
		ele.AskVol3 = ele.AskLevels[2].Vol
		ele.AskVol4 = ele.AskLevels[3].Vol
		ele.AskVol5 = ele.AskLevels[4].Vol

		if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Unknown); err != nil {
			return err
		}
		pos += 2
		pos += 4

		var riseSpeed int16
		if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &riseSpeed); err != nil {
			return err
		}
		pos += 2
		ele.Rate = float64(riseSpeed) / 100.0
		if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Active2); err != nil {
			return err
		}
		pos += 2

		obj.reply.List = append(obj.reply.List, ele)
	}

	return nil
}

func (obj *GetSecurityQuotes) Reply() *GetSecurityQuotesReply {
	return obj.reply
}

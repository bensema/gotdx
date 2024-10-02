package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type GetSecurityQuotes struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityQuotesRequest
	reply      *GetSecurityQuotesReply

	contentHex string
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
	Market         uint8   // 市场
	Code           string  // 代码
	Active1        uint16  // 活跃度
	Price          float64 // 现价
	LastClose      float64 // 昨收
	Open           float64 // 开盘
	High           float64 // 最高
	Low            float64 // 最低
	ServerTime     string  // 时间
	ReversedBytes0 int     // 保留(时间 ServerTime)
	ReversedBytes1 int     // 保留
	Vol            int     // 总量
	CurVol         int     // 现量
	Amount         float64 // 总金额
	SVol           int     // 内盘
	BVol           int     // 外盘
	ReversedBytes2 int     // 保留
	ReversedBytes3 int     // 保留
	BidLevels      []Level
	AskLevels      []Level
	Bid1           float64
	Ask1           float64
	BidVol1        int
	AskVol1        int
	Bid2           float64
	Ask2           float64
	BidVol2        int
	AskVol2        int
	Bid3           float64
	Ask3           float64
	BidVol3        int
	AskVol3        int
	Bid4           float64
	Ask4           float64
	BidVol4        int
	AskVol4        int
	Bid5           float64
	Ask5           float64
	BidVol5        int
	AskVol5        int
	ReversedBytes4 uint16  // 保留
	ReversedBytes5 int     // 保留
	ReversedBytes6 int     // 保留
	ReversedBytes7 int     // 保留
	ReversedBytes8 int     // 保留
	Rate           float64 // 涨速
	Active2        uint16  // 活跃度
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
	obj.contentHex = "0500000000000000"
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
	b, err := hex.DecodeString(obj.contentHex)
	buf.Write(b)

	err = binary.Write(buf, binary.LittleEndian, uint16(len(obj.request.StockList)))

	for _, stock := range obj.request.StockList {
		//code, _ := hex.DecodeString(stock.Code)
		//code := []byte{}
		code := make([]byte, 6)
		copy(code, stock.Code)
		tmp := []byte{stock.Market}
		tmp = append(tmp, code...)
		buf.Write(tmp)
	}

	return buf.Bytes(), err
}

func (obj *GetSecurityQuotes) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	//fmt.Println(hex.EncodeToString(data))
	pos := 0

	pos += 2 // 跳过两个字节
	binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count)
	pos += 2
	for index := uint16(0); index < obj.reply.Count; index++ {
		ele := SecurityQuote{}
		binary.Read(bytes.NewBuffer(data[pos:pos+1]), binary.LittleEndian, &ele.Market)
		pos += 1
		var code [6]byte
		binary.Read(bytes.NewBuffer(data[pos:pos+6]), binary.LittleEndian, &code)
		//enc := mahonia.NewDecoder("gbk")
		//ele.Code = enc.ConvertString(string(code[:]))
		ele.Code = Utf8ToGbk(code[:])
		pos += 6
		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Active1)
		pos += 2

		price := getprice(data, &pos)
		ele.Price = obj.getPrice(price, 0)
		ele.LastClose = obj.getPrice(price, getprice(data, &pos))
		ele.Open = obj.getPrice(price, getprice(data, &pos))
		ele.High = obj.getPrice(price, getprice(data, &pos))
		ele.Low = obj.getPrice(price, getprice(data, &pos))

		ele.ReversedBytes0 = getprice(data, &pos)
		ele.ServerTime = fmt.Sprintf("%d", ele.ReversedBytes0)
		ele.ReversedBytes1 = getprice(data, &pos)

		ele.Vol = getprice(data, &pos)
		ele.CurVol = getprice(data, &pos)

		var amountraw uint32
		binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &amountraw)
		pos += 4
		ele.Amount = getvolume(int(amountraw))

		ele.SVol = getprice(data, &pos)
		ele.BVol = getprice(data, &pos)

		ele.ReversedBytes2 = getprice(data, &pos)
		ele.ReversedBytes3 = getprice(data, &pos)

		for i := 0; i < 5; i++ {
			bidele := Level{Price: obj.getPrice(getprice(data, &pos), price)}
			offerele := Level{Price: obj.getPrice(getprice(data, &pos), price)}
			bidele.Vol = getprice(data, &pos)
			offerele.Vol = getprice(data, &pos)
			ele.BidLevels = append(ele.BidLevels, bidele)
			ele.AskLevels = append(ele.AskLevels, offerele)
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

		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.ReversedBytes4)
		pos += 2
		ele.ReversedBytes5 = getprice(data, &pos)
		ele.ReversedBytes6 = getprice(data, &pos)
		ele.ReversedBytes7 = getprice(data, &pos)
		ele.ReversedBytes8 = getprice(data, &pos)

		var reversedbytes9 int16
		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &reversedbytes9)
		pos += 2
		ele.Rate = float64(reversedbytes9) / 100.0
		binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &ele.Active2)
		pos += 2

		obj.reply.List = append(obj.reply.List, ele)
	}
	return nil
}

func (obj *GetSecurityQuotes) Reply() *GetSecurityQuotesReply {
	return obj.reply
}

func (obj *GetSecurityQuotes) getPrice(price int, diff int) float64 {
	return float64(price+diff) / 100.0
}

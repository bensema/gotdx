package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type GetHistoryMinuteTimeData struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryMinuteTimeDataRequest
	reply      *GetHistoryMinuteTimeDataReply
}

type GetHistoryMinuteTimeDataRequest struct {
	Date   int32   // 交易日期，协议中使用负值编码。
	Market uint8   // 市场代码。
	Code   [6]byte // 证券代码。
}

type GetHistoryMinuteTimeDataReply struct {
	Count uint16                  // 返回条数。
	List  []HistoryMinuteTimeData // 历史分时数据。
}

type HistoryMinuteTimeData struct {
	Price float64 // 成交价。
	Avg   float64 // 均价。
	Vol   int     // 成交量。
}

func NewGetHistoryMinuteTimeData(req *GetHistoryMinuteTimeDataRequest) *GetHistoryMinuteTimeData {
	obj := new(GetHistoryMinuteTimeData)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetHistoryMinuteTimeDataRequest)
	obj.reply = new(GetHistoryMinuteTimeDataReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_HISTORYMINUTETIMEDATE
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetHistoryMinuteTimeData) applyRequest(req *GetHistoryMinuteTimeDataRequest) {
	if req.Date > 0 {
		req.Date = -req.Date
	}
	obj.request = req
}

func (obj *GetHistoryMinuteTimeData) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x0d
	obj.reqHeader.PkgLen2 = 0x0d

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetHistoryMinuteTimeData) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 校验最小报文长度，避免空或超短数据触发越界 panic。
	if len(data) < 10 {
		return fmt.Errorf("invalid history minute response length: %d", len(data))
	}

	pos := 0
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 10 // count + 2 unknown uint32 fields

	startPrice := 0
	startAvg := 0
	unit := baseUnit(string(obj.request.Code[:]))
	for index := uint16(0); index < obj.reply.Count; index++ {
		price, ok := readPriceField(data, &pos)
		if !ok {
			return fmt.Errorf("truncated history minute response: pos=%d len=%d", pos, len(data))
		}
		avg, ok := readPriceField(data, &pos)
		if !ok {
			return fmt.Errorf("truncated history minute response: pos=%d len=%d", pos, len(data))
		}
		vol, ok := readPriceField(data, &pos)
		if !ok {
			return fmt.Errorf("truncated history minute response: pos=%d len=%d", pos, len(data))
		}

		if startPrice != 0 {
			price += startPrice
		}
		if startAvg != 0 {
			avg += startAvg
		}

		obj.reply.List = append(obj.reply.List, HistoryMinuteTimeData{
			Price: float64(price) / unit,
			Avg:   float64(avg) / (unit * 100),
			Vol:   vol,
		})

		if startPrice == 0 {
			startPrice = price
		}
		if startAvg == 0 {
			startAvg = avg
		}
	}

	return nil
}

// readPriceField 安全读取一条变长价格字段；字段不完整（含续字节截断）时返回 false。
func readPriceField(data []byte, pos *int) (int, bool) {
	i := *pos
	for {
		if i >= len(data) {
			return 0, false
		}
		// 变长编码的续位：0x80 表示后面还有字节，扫描至末位为 0 的字节结束。
		if data[i]&0x80 == 0 {
			break
		}
		i++
	}
	return getprice(data, pos), true
}

func (obj *GetHistoryMinuteTimeData) Response() *GetHistoryMinuteTimeDataReply {
	return obj.reply
}

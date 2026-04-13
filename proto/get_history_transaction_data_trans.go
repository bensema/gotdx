package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

type GetHistoryTransactionDataWithTrans struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetHistoryTransactionDataRequest
	reply      *GetHistoryTransactionDataWithTransReply
}

type GetHistoryTransactionDataWithTransReply struct {
	Count    uint16                            // 返回条数。
	PreClose float64                           // 昨收价。
	List     []HistoryTransactionDataWithTrans // 带方向的历史逐笔成交。
}

type HistoryTransactionDataWithTrans struct {
	Time   string  // 成交时间。
	Price  float64 // 成交价。
	Vol    int     // 成交量。
	Num    int     // 笔数或委托笔数。
	Action string  // 成交方向，如 BUY/SELL/NEUTRAL。
}

func NewGetHistoryTransactionDataWithTrans(req *GetHistoryTransactionDataRequest) *GetHistoryTransactionDataWithTrans {
	obj := &GetHistoryTransactionDataWithTrans{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetHistoryTransactionDataRequest),
		reply:      new(GetHistoryTransactionDataWithTransReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_TRANSACTIONDATA_TRANS
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetHistoryTransactionDataWithTrans) applyRequest(req *GetHistoryTransactionDataRequest) {
	obj.request = req
}

func (obj *GetHistoryTransactionDataWithTrans) BuildRequest() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 0x12
	obj.reqHeader.PkgLen2 = 0x12
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, obj.reqHeader); err != nil {
		return nil, err
	}
	err := binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetHistoryTransactionDataWithTrans) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 6 {
		return fmt.Errorf("invalid history transaction with trans response length: %d", len(data))
	}
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])
	obj.reply.PreClose = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[2:6])))
	pos := 6
	lastPrice := 0
	for i := uint16(0); i < obj.reply.Count; i++ {
		hour, minute := gettime(data, &pos)
		priceDiff := getprice(data, &pos)
		vol := getprice(data, &pos)
		num := getprice(data, &pos)
		if pos+2 > len(data) {
			return fmt.Errorf("invalid history transaction with trans item %d", i)
		}
		actionCode := binary.LittleEndian.Uint16(data[pos : pos+2])
		pos += 2
		lastPrice += priceDiff
		item := HistoryTransactionDataWithTrans{
			Time:  fmt.Sprintf("%02d:%02d", hour, minute),
			Price: float64(lastPrice) / baseUnit(string(obj.request.Code[:])),
			Vol:   vol,
			Num:   num,
		}
		switch actionCode {
		case 0:
			item.Action = "BUY"
		case 1:
			item.Action = "SELL"
		case 2:
			item.Action = "NEUTRAL"
		default:
			item.Action = fmt.Sprintf("%d", actionCode)
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	return nil
}

func (obj *GetHistoryTransactionDataWithTrans) Response() *GetHistoryTransactionDataWithTransReply {
	return obj.reply
}

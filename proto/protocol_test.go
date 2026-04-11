package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func encodePrice(value int) []byte {
	sign := value < 0
	if sign {
		value = -value
	}

	first := byte(value & 0x3f)
	value >>= 6
	if sign {
		first |= 0x40
	}
	if value > 0 {
		first |= 0x80
	}

	out := []byte{first}
	for value > 0 {
		part := byte(value & 0x7f)
		value >>= 7
		if value > 0 {
			part |= 0x80
		}
		out = append(out, part)
	}

	return out
}

func mustSerialize(t *testing.T, msg Msg) []byte {
	t.Helper()
	raw, err := msg.Serialize()
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	return raw
}

func readReqHeader(t *testing.T, raw []byte) ReqHeader {
	t.Helper()
	var header ReqHeader
	if err := binary.Read(bytes.NewReader(raw[:12]), binary.LittleEndian, &header); err != nil {
		t.Fatalf("read header failed: %v", err)
	}
	return header
}

func TestGetSecurityCountSerializeUsesTodayDate(t *testing.T) {
	msg := NewGetSecurityCount()
	msg.SetParams(&GetSecurityCountRequest{Market: 1})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_SECURITYCOUNT {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetSecurityCountRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}

	if req.Market != 1 {
		t.Fatalf("unexpected market: %d", req.Market)
	}
	if req.Date != todayDate() {
		t.Fatalf("unexpected date: got %d want %d", req.Date, todayDate())
	}
}

func TestGetSecurityListSerializeAndDeserialize(t *testing.T) {
	msg := NewGetSecurityList()
	msg.SetParams(&GetSecurityListRequest{Market: 1, Start: 5})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_SECURITYLIST {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetSecurityListRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Start != 5 {
		t.Fatalf("unexpected start: %d", req.Start)
	}
	if req.Count != 1600 {
		t.Fatalf("unexpected count: %d", req.Count)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := [6]byte{'6', '0', '0', '0', '0', '0'}
	if err := binary.Write(buf, binary.LittleEndian, code); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(100)); err != nil {
		t.Fatal(err)
	}
	name := [16]byte{'T', 'E', 'S', 'T'}
	if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(1.5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int8(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.34)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(8)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 1 {
		t.Fatalf("unexpected reply count: %d", reply.Count)
	}
	stock := reply.List[0]
	if stock.Code != "600000" {
		t.Fatalf("unexpected code: %q", stock.Code)
	}
	if stock.Vol != 100 || stock.VolUnit != 100 {
		t.Fatalf("unexpected vol: %d/%d", stock.Vol, stock.VolUnit)
	}
	if stock.Name != "TEST" {
		t.Fatalf("unexpected name: %q", stock.Name)
	}
	if stock.DecimalPoint != 2 {
		t.Fatalf("unexpected decimal point: %d", stock.DecimalPoint)
	}
	if math.Abs(stock.PreClose-12.34) > 0.001 {
		t.Fatalf("unexpected pre_close: %f", stock.PreClose)
	}
	if stock.Unknown2 != 7 || stock.Unknown3 != 8 {
		t.Fatalf("unexpected unknowns: %d/%d", stock.Unknown2, stock.Unknown3)
	}
}

func TestGetMinuteTimeDataSerializeAndDeserialize(t *testing.T) {
	msg := NewGetMinuteTimeData()
	msg.SetParams(&GetMinuteTimeDataRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
		Start:  3,
		Count:  10,
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_MINUTETIMEDATA {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetMinuteTimeDataRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Start != 3 || req.Count != 10 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1000))
	buf.Write(encodePrice(100000))
	buf.Write(encodePrice(10))
	buf.Write(encodePrice(5))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(20))

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 2 {
		t.Fatalf("unexpected count: %d", reply.Count)
	}
	if math.Abs(reply.List[0].Price-10.0) > 0.001 || math.Abs(reply.List[0].Avg-10.0) > 0.001 {
		t.Fatalf("unexpected first point: %+v", reply.List[0])
	}
	if math.Abs(reply.List[1].Price-10.05) > 0.001 || math.Abs(reply.List[1].Avg-10.01) > 0.001 {
		t.Fatalf("unexpected second point: %+v", reply.List[1])
	}
}

func TestGetHistoryMinuteTimeDataSerializeAndDeserialize(t *testing.T) {
	msg := NewGetHistoryMinuteTimeData()
	msg.SetParams(&GetHistoryMinuteTimeDataRequest{
		Date:   20240531,
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_HISTORYMINUTETIMEDATE {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetHistoryMinuteTimeDataRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Date != -20240531 {
		t.Fatalf("unexpected request date: %d", req.Date)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1000))
	buf.Write(encodePrice(100000))
	buf.Write(encodePrice(20))

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 1 {
		t.Fatalf("unexpected count: %d", reply.Count)
	}
	if math.Abs(reply.List[0].Price-10.0) > 0.001 || math.Abs(reply.List[0].Avg-10.0) > 0.001 {
		t.Fatalf("unexpected point: %+v", reply.List[0])
	}
}

func TestGetSecurityBarsSerializeAndDeserialize(t *testing.T) {
	msg := NewGetSecurityBars()
	msg.SetParams(&GetSecurityBarsRequest{
		Market:   1,
		Code:     [6]byte{'6', '0', '0', '0', '0', '0'},
		Category: KLINE_TYPE_DAILY,
		Times:    2,
		Start:    3,
		Count:    4,
		Adjust:   1,
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_SECURITYBARS {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetSecurityBarsRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Times != 2 || req.Adjust != 1 || req.Start != 3 || req.Count != 4 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20240531)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(12345))
	buf.Write(encodePrice(12355))
	buf.Write(encodePrice(12400))
	buf.Write(encodePrice(12200))
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(3456.5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(7890.25)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(9)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 1 {
		t.Fatalf("unexpected count: %d", reply.Count)
	}
	bar := reply.List[0]
	if bar.DateTime != "2024-05-31 15:00:00" {
		t.Fatalf("unexpected datetime: %s", bar.DateTime)
	}
	if math.Abs(bar.Open-12.345) > 0.0001 || math.Abs(bar.Close-12.355) > 0.0001 {
		t.Fatalf("unexpected prices: %+v", bar)
	}
	if math.Abs(bar.Vol-3456.5) > 0.001 || math.Abs(bar.Amount-7890.25) > 0.001 {
		t.Fatalf("unexpected volume/amount: %+v", bar)
	}
	if bar.UpCount != 7 || bar.DownCount != 9 {
		t.Fatalf("unexpected counts: %+v", bar)
	}
}

func TestGetSecurityQuotesSerializeAndDeserialize(t *testing.T) {
	msg := NewGetSecurityQuotes()
	msg.SetParams(&GetSecurityQuotesRequest{
		StockList: []Stock{{Market: 1, Code: "600000"}},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_SECURITYQUOTES {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(-10))
	buf.Write(encodePrice(-5))
	buf.Write(encodePrice(20))
	buf.Write(encodePrice(-30))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(15))
	buf.Write(encodePrice(10000))
	buf.Write(encodePrice(500))
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(123456.5)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(300))
	buf.Write(encodePrice(400))
	buf.Write(encodePrice(50))
	buf.Write(encodePrice(60))
	for i := 0; i < 5; i++ {
		buf.Write(encodePrice(-(i + 1)))
		buf.Write(encodePrice(i + 1))
		buf.Write(encodePrice(100 + i))
		buf.Write(encodePrice(200 + i))
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(3)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, [4]byte{}); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(25)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(9)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 1 {
		t.Fatalf("unexpected count: %d", reply.Count)
	}
	quote := reply.List[0]
	if quote.Code != "600000" {
		t.Fatalf("unexpected code: %q", quote.Code)
	}
	if math.Abs(quote.Close-12.34) > 0.001 || math.Abs(quote.PreClose-12.24) > 0.001 {
		t.Fatalf("unexpected prices: %+v", quote)
	}
	if math.Abs(quote.Open-12.29) > 0.001 || math.Abs(quote.High-12.54) > 0.001 || math.Abs(quote.Low-12.04) > 0.001 {
		t.Fatalf("unexpected OHLC: %+v", quote)
	}
	if quote.ServerTime != "00:00:00.000" {
		t.Fatalf("unexpected server time: %s", quote.ServerTime)
	}
	if math.Abs(quote.Bid1-12.33) > 0.001 || math.Abs(quote.Ask1-12.35) > 0.001 {
		t.Fatalf("unexpected level1: %+v", quote)
	}
	if quote.BidVol1 != 100 || quote.AskVol1 != 200 {
		t.Fatalf("unexpected level1 vol: %+v", quote)
	}
	if math.Abs(quote.Rate-0.25) > 0.001 {
		t.Fatalf("unexpected rate: %+v", quote)
	}
}

func TestGetIndexMomentumSerializeAndDeserialize(t *testing.T) {
	msg := NewGetIndexMomentum()
	msg.SetParams(&GetIndexMomentumRequest{
		Market: 1,
		Code:   [6]byte{'0', '0', '0', '0', '0', '1'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_INDEXMOMENTUM {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(3)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1))
	buf.Write(encodePrice(2))
	buf.Write(encodePrice(-1))

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 3 {
		t.Fatalf("unexpected count: %d", reply.Count)
	}
	want := []int{1, 3, 2}
	for i, value := range want {
		if reply.Values[i] != value {
			t.Fatalf("unexpected values: %+v", reply.Values)
		}
	}
}

func TestGetChartSamplingSerializeAndDeserialize(t *testing.T) {
	msg := NewGetChartSampling()
	msg.SetParams(&GetChartSamplingRequest{
		Market: 1,
		Code:   [6]byte{'0', '0', '0', '0', '0', '1'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_CHARTSAMPLING {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("000001")); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(make([]byte, 26)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.34)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.35)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.36)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Market != 1 || reply.Code != "000001" || reply.Count != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if math.Abs(reply.PreClose-12.34) > 0.001 || math.Abs(reply.Prices[0]-12.35) > 0.001 || math.Abs(reply.Prices[1]-12.36) > 0.001 {
		t.Fatalf("unexpected prices: %+v", reply)
	}
}

func TestGetAuctionSerializeAndDeserialize(t *testing.T) {
	msg := NewGetAuction()
	msg.SetParams(&GetAuctionRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
		Start:  2,
		Count:  3,
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_AUCTION {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetAuctionRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Mode != 3 || req.Start != 2 || req.Count != 3 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(9*60+25)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.34)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(1000)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(200)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(30)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	item := msg.Reply().List[0]
	if item.Time != "09:25:30" || math.Abs(item.Price-12.34) > 0.001 || item.Matched != 1000 || item.Unmatched != 200 {
		t.Fatalf("unexpected auction item: %+v", item)
	}
}

func TestGetUnusualSerializeAndDeserialize(t *testing.T) {
	msg := NewGetUnusual()
	msg.SetParams(&GetUnusualRequest{
		Market: 1,
		Start:  10,
		Count:  20,
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_UNUSUAL {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	record := new(bytes.Buffer)
	if err := binary.Write(record, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := record.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint8(0x04)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint16(12)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, math.Float32bits(0.0123)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, math.Float32bits(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, math.Float32bits(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint8(9)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(record, binary.LittleEndian, uint16(3015)); err != nil {
		t.Fatal(err)
	}
	if record.Len() != 32 {
		t.Fatalf("unexpected record size: %d", record.Len())
	}
	if _, err := buf.Write(record.Bytes()); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	item := msg.Reply().List[0]
	if item.Code != "600000" || item.Time != "09:30:15" || item.Desc != "加速拉升" || item.Value != "1.23%" {
		t.Fatalf("unexpected unusual item: %+v", item)
	}
}

func TestGetHistoryOrdersSerializeAndDeserialize(t *testing.T) {
	msg := NewGetHistoryOrders()
	msg.SetParams(&GetHistoryOrdersRequest{
		Date:   20240531,
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_HISTORYORDERS {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.34)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(7))
	buf.Write(encodePrice(100))

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Count != 1 || math.Abs(reply.PreClose-12.34) > 0.001 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if math.Abs(item.Price-12.34) > 0.001 || item.Unknown != 7 || item.Vol != 100 {
		t.Fatalf("unexpected history order: %+v", item)
	}
}

func TestGetTopBoardSerializeAndDeserialize(t *testing.T) {
	msg := NewGetTopBoard()
	msg.SetParams(&GetTopBoardRequest{
		Category: 6,
		Size:     1,
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_TOPBOARD {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 9; i++ {
		if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
			t.Fatal(err)
		}
		if _, err := buf.Write([]byte("600000")); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.34)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(1.23)); err != nil {
			t.Fatal(err)
		}
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Size != 1 || len(reply.Increase) != 1 || len(reply.Turnover) != 1 {
		t.Fatalf("unexpected reply sizes: %+v", reply)
	}
	if reply.Increase[0].Code != "600000" || math.Abs(reply.Increase[0].Price-12.34) > 0.001 || math.Abs(reply.Increase[0].Value-1.23) > 0.001 {
		t.Fatalf("unexpected board item: %+v", reply.Increase[0])
	}
}

func TestGetIndexInfoSerializeAndDeserialize(t *testing.T) {
	msg := NewGetIndexInfo()
	msg.SetParams(&GetIndexInfoRequest{
		Market: 1,
		Code:   [6]byte{'0', '0', '0', '0', '0', '1'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_INDEXINFO {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("000001")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(-10))
	buf.Write(encodePrice(5))
	buf.Write(encodePrice(20))
	buf.Write(encodePrice(-30))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(15))
	buf.Write(encodePrice(10000))
	buf.Write(encodePrice(500))
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(123456.5)); err != nil {
		t.Fatal(err)
	}
	extras := []int{1, 2, 60, 3, 4, 5, 88, 99, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	for _, value := range extras {
		buf.Write(encodePrice(value))
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(7))
	buf.Write(encodePrice(100))

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.OrderCount != 1 || reply.Code != "000001" || reply.Active != 7 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if math.Abs(reply.Close-12.34) > 0.001 || math.Abs(reply.PreClose-12.24) > 0.001 || math.Abs(reply.Diff-0.10) > 0.001 {
		t.Fatalf("unexpected price fields: %+v", reply)
	}
	if math.Abs(reply.Open-12.39) > 0.001 || math.Abs(reply.High-12.54) > 0.001 || math.Abs(reply.Low-12.04) > 0.001 {
		t.Fatalf("unexpected ohl fields: %+v", reply)
	}
	if reply.OpenAmount != 60 || reply.UpCount != 88 || reply.DownCount != 99 || len(reply.Orders) != 1 {
		t.Fatalf("unexpected counts/orders: %+v", reply)
	}
	if math.Abs(reply.Orders[0].Price-12.34) > 0.001 || reply.Orders[0].Unknown != 7 || reply.Orders[0].Vol != 100 {
		t.Fatalf("unexpected order: %+v", reply.Orders[0])
	}
}

func TestGetQuotesListSerializeAndDeserialize(t *testing.T) {
	msg := NewGetQuotesList()
	msg.SetParams(&GetQuotesListRequest{
		Category:    6,
		SortType:    6,
		Start:       1,
		Count:       2,
		SortReverse: 1,
		Filter:      4,
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_QUOTESLIST {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(-10))
	buf.Write(encodePrice(5))
	buf.Write(encodePrice(20))
	buf.Write(encodePrice(-30))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(15))
	buf.Write(encodePrice(10000))
	buf.Write(encodePrice(500))
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(123456.5)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(300))
	buf.Write(encodePrice(400))
	buf.Write(encodePrice(50))
	buf.Write(encodePrice(60))
	buf.Write(encodePrice(-1))
	buf.Write(encodePrice(1))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(200))
	if err := binary.Write(buf, binary.LittleEndian, uint16(3)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(25)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(123)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(456.5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(78)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(make([]byte, 10)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(4.5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(5.5)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(make([]byte, 24)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(9)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if reply.Block != 5 || reply.Count != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.Code != "600000" || math.Abs(item.Close-12.34) > 0.001 || math.Abs(item.PreClose-12.24) > 0.001 {
		t.Fatalf("unexpected item prices: %+v", item)
	}
	if math.Abs(item.RiseSpeed-0.25) > 0.001 || math.Abs(item.ShortTurnover-1.23) > 0.001 || math.Abs(item.OpeningRush-0.78) > 0.001 {
		t.Fatalf("unexpected item rates: %+v", item)
	}
	if math.Abs(item.Min2Amount-456.5) > 0.001 || math.Abs(item.VolRiseSpeed-4.5) > 0.001 || math.Abs(item.Depth-5.5) > 0.001 {
		t.Fatalf("unexpected float tails: %+v", item)
	}
}

func TestGetQuotesSerializeAndDeserialize(t *testing.T) {
	msg := NewGetQuotes()
	msg.SetParams(&GetQuotesRequest{
		Stocks: []Stock{{Market: 1, Code: "600000"}},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_QUOTES {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	if binary.LittleEndian.Uint16(raw[12:14]) != 5 || binary.LittleEndian.Uint16(raw[20:22]) != 1 {
		t.Fatalf("unexpected request payload")
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(-10))
	buf.Write(encodePrice(5))
	buf.Write(encodePrice(20))
	buf.Write(encodePrice(-30))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(15))
	buf.Write(encodePrice(10000))
	buf.Write(encodePrice(500))
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(123456.5)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(300))
	buf.Write(encodePrice(400))
	buf.Write(encodePrice(50))
	buf.Write(encodePrice(60))
	buf.Write(encodePrice(-1))
	buf.Write(encodePrice(1))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(200))
	if err := binary.Write(buf, binary.LittleEndian, uint16(3)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(25)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(123)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(456.5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int16(78)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(make([]byte, 10)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(4.5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(5.5)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(make([]byte, 24)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(9)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if msg.Reply().Count != 1 || msg.Reply().List[0].Code != "600000" {
		t.Fatalf("unexpected reply: %+v", msg.Reply())
	}
}

func TestGetVolumeProfileSerializeAndDeserialize(t *testing.T) {
	msg := NewGetVolumeProfile()
	msg.SetParams(&GetVolumeProfileRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_VOLUMEPROFILE {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(-10))
	buf.Write(encodePrice(5))
	buf.Write(encodePrice(20))
	buf.Write(encodePrice(-30))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(15))
	buf.Write(encodePrice(10000))
	buf.Write(encodePrice(500))
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(123456.5)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(300))
	buf.Write(encodePrice(400))
	buf.Write(encodePrice(50))
	buf.Write(encodePrice(60))
	for i := 0; i < 3; i++ {
		buf.Write(encodePrice(i + 1))
		buf.Write(encodePrice(i + 2))
		buf.Write(encodePrice(100 + i))
		buf.Write(encodePrice(200 + i))
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(42)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(40))
	buf.Write(encodePrice(60))
	buf.Write(encodePrice(2))
	buf.Write(encodePrice(50))
	buf.Write(encodePrice(20))
	buf.Write(encodePrice(30))

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	reply := msg.Reply()
	if reply.Count != 2 || reply.Code != "600000" || len(reply.VolProfiles) != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if math.Abs(reply.Close-12.34) > 0.001 || math.Abs(reply.VolProfiles[0].Price-12.34) > 0.001 || math.Abs(reply.VolProfiles[1].Price-12.36) > 0.001 {
		t.Fatalf("unexpected profile prices: %+v", reply)
	}
}

func TestGetCompanyCategorySerializeAndDeserialize(t *testing.T) {
	msg := NewGetCompanyCategory()
	msg.SetParams(&GetCompanyCategoryRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_COMPANYCATEGORY {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	name := [64]byte{'F', '1', '0'}
	file := [80]byte{'t', 'e', 's', 't', '.', 't', 'x', 't'}
	if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, file); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(12)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(34)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	reply := msg.Reply()
	if reply.Count != 1 || reply.Categories[0].Name != "F10" || reply.Categories[0].Filename != "test.txt" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestGetCompanyCategoryTrimsFixedStringGarbage(t *testing.T) {
	msg := NewGetCompanyCategory()

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}

	name := make([]byte, 64)
	copy(name, []byte{0xd7, 0xca, 0xbd, 0xf0, 0xb6, 0xaf, 0xcf, 0xf2})
	name[8] = 0x00
	copy(name[9:], []byte{0xff, 0xfe, '8'})

	file := make([]byte, 80)
	copy(file, []byte("000001.txt"))
	file[10] = 0x00
	copy(file[11:], []byte{0x80, 0x81, 'O', 0xa4})

	if _, err := buf.Write(name); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write(file); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(12)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(34)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}

	reply := msg.Reply()
	if got := reply.Categories[0].Name; got != "资金动向" {
		t.Fatalf("unexpected category name: %q", got)
	}
	if got := reply.Categories[0].Filename; got != "000001.txt" {
		t.Fatalf("unexpected category filename: %q", got)
	}
}

func TestGetCompanyContentSerializeAndDeserialize(t *testing.T) {
	msg := NewGetCompanyContent()
	req := GetCompanyContentRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
		Start:  12,
		Length: 4,
	}
	copy(req.Filename[:], "test.txt")
	msg.SetParams(&req)

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_COMPANYCONTENT {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(4)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("ABCD")); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if msg.Reply().Content != "ABCD" || msg.Reply().Code != "600000" {
		t.Fatalf("unexpected reply: %+v", msg.Reply())
	}
}

func TestGetFinanceInfoSerializeAndDeserialize(t *testing.T) {
	msg := NewGetFinanceInfo()
	msg.SetParams(&GetFinanceInfoRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_FINANCEINFO {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, float32(123.45)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(3)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20240531)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20100101)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		if err := binary.Write(buf, binary.LittleEndian, float32(i+1)); err != nil {
			t.Fatal(err)
		}
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	reply := msg.Reply()
	if reply.Code != "600000" || reply.Province != 2 || reply.Industry != 3 || math.Abs(float64(reply.FloatShares-123.45)) > 0.001 {
		t.Fatalf("unexpected finance reply: %+v", reply)
	}
	if reply.TotalShares != 1 || reply.Reserved2 != 30 {
		t.Fatalf("unexpected float mapping: %+v", reply)
	}
}

func TestGetXDXRInfoSerializeAndDeserialize(t *testing.T) {
	msg := NewGetXDXRInfo()
	msg.SetParams(&GetXDXRInfoRequest{
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
	})

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_XDXRINFO {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20240531)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(1)); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float32{1.1, 2.2, 3.3, 4.4} {
		if err := binary.Write(buf, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	reply := msg.Reply()
	if reply.Code != "600000" || reply.Count != 1 || reply.List[0].Name != "除权除息" {
		t.Fatalf("unexpected xdxr reply: %+v", reply)
	}
	if reply.List[0].Fenhong == nil || math.Abs(float64(*reply.List[0].Fenhong-1.1)) > 0.001 {
		t.Fatalf("unexpected xdxr data: %+v", reply.List[0])
	}
}

func TestGetFileMetaSerializeAndDeserialize(t *testing.T) {
	msg := NewGetFileMeta()
	req := GetFileMetaRequest{}
	copy(req.Filename[:], "block.dat")
	msg.SetParams(&req)

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_BLOCKINFOMETA {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(123)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, byte(1)); err != nil {
		t.Fatal(err)
	}
	hash := [32]byte{'h', 'a', 's', 'h'}
	if err := binary.Write(buf, binary.LittleEndian, hash); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, byte(2)); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if msg.Reply().Size != 123 || msg.Reply().Unknown1 != 1 || msg.Reply().Unknown2 != 2 {
		t.Fatalf("unexpected meta reply: %+v", msg.Reply())
	}
}

func TestDownloadFileSerializeAndDeserialize(t *testing.T) {
	msg := NewDownloadFile()
	req := DownloadFileRequest{Start: 10, Size: 20}
	copy(req.Filename[:], "block.dat")
	msg.SetParams(&req)

	raw := mustSerialize(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_BLOCKINFO {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(4)); err != nil {
		t.Fatal(err)
	}
	if _, err := buf.Write([]byte("DATA")); err != nil {
		t.Fatal(err)
	}

	if err := msg.UnSerialize(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if msg.Reply().Size != 4 || string(msg.Reply().Data) != "DATA" {
		t.Fatalf("unexpected download reply: %+v", msg.Reply())
	}
}

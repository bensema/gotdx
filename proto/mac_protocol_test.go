package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func assertNearFloat64(t *testing.T, got float64, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("unexpected %s: got=%.6f want=%.6f", label, got, want)
	}
}

func writeMACFloat32(t *testing.T, buf *bytes.Buffer, value float32) {
	t.Helper()
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(value)); err != nil {
		t.Fatalf("write float32 failed: %v", err)
	}
}

func TestExchangeMACBoardCode(t *testing.T) {
	tests := []struct {
		symbol string
		want   uint32
	}{
		{symbol: "880761", want: 20761},
		{symbol: "HK0281", want: 20281},
		{symbol: "US0401", want: 30401},
		{symbol: "399372", want: 30372},
		{symbol: "000686", want: 31686},
	}

	for _, tt := range tests {
		got, err := ExchangeMACBoardCode(tt.symbol)
		if err != nil {
			t.Fatalf("ExchangeMACBoardCode(%q) failed: %v", tt.symbol, err)
		}
		if got != tt.want {
			t.Fatalf("ExchangeMACBoardCode(%q) = %d, want %d", tt.symbol, got, tt.want)
		}
	}
}

func TestMACBoardCountBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACBoardCount(&MACBoardListRequest{BoardType: 5})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXBOARDLIST {
		t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACBoardListRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.BoardType != 5 || req.PageSize != 150 || req.SortOrder != 1 || req.One != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	payload := []byte{0x2c, 0x01, 0x2f, 0x02}
	if err := msg.ParseResponse(&RespHeader{}, payload); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.CountAll != 300 || reply.Total != 559 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestMACBoardListParseResponse(t *testing.T) {
	msg := NewMACBoardList(nil)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(559)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := [6]byte{'8', '8', '0', '0', '0', '1'}
	if err := binary.Write(buf, binary.LittleEndian, code); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 16))
	name := make([]byte, 44)
	copy(name, "Coal")
	buf.Write(name)
	writeMACFloat32(t, buf, 10.5)
	writeMACFloat32(t, buf, 0.8)
	writeMACFloat32(t, buf, 10.0)
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	symbolCode := [6]byte{'0', '0', '0', '0', '0', '1'}
	if err := binary.Write(buf, binary.LittleEndian, symbolCode); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 16))
	symbolName := make([]byte, 44)
	copy(symbolName, "PingAn")
	buf.Write(symbolName)
	writeMACFloat32(t, buf, 12.3)
	writeMACFloat32(t, buf, 0.1)
	writeMACFloat32(t, buf, 12.0)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.Code != "880001" || item.Name != "Coal" || item.SymbolCode != "000001" || item.SymbolName != "PingAn" {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestMACBoardMembersBuildRequestAndParseResponse(t *testing.T) {
	boardCode, err := ExchangeMACBoardCode("880761")
	if err != nil {
		t.Fatal(err)
	}
	msg := NewMACBoardMembers(&MACBoardMembersRequest{
		BoardCode: boardCode,
		Start:     10,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACBOARDMEMBERS {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACBoardMembersRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.BoardCode != boardCode || req.PageSize != 80 || req.SortType != 14 || req.Start != 10 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	buf.Write(make([]byte, 16))
	name := [4]byte{'B', 'K', '0', '1'}
	if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(123)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	stockCode := [6]byte{'6', '0', '0', '0', '0', '0'}
	if err := binary.Write(buf, binary.LittleEndian, stockCode); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 16))
	stockName := [16]byte{'B', 'A', 'N', 'K'}
	if err := binary.Write(buf, binary.LittleEndian, stockName); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 28))

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || reply.Total != 123 || len(reply.Stocks) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.Stocks[0].Symbol != "600000" || reply.Stocks[0].Name != "BANK" {
		t.Fatalf("unexpected stock: %+v", reply.Stocks[0])
	}
}

func TestMACBoardMembersQuotesBuildRequestAndParseResponse(t *testing.T) {
	boardCode, err := ExchangeMACBoardCode("880761")
	if err != nil {
		t.Fatal(err)
	}
	msg := NewMACBoardMembersQuotes(&MACBoardMembersQuotesRequest{BoardCode: boardCode})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACBOARDMEMBERS {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACBoardMembersQuotesRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.BoardCode != boardCode || req.PageSize != 80 || req.SortOrder != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}
	if req.Extra[1] != 0xff || req.Extra[5] != 0x3f {
		t.Fatalf("unexpected extra selector: %#v", req.Extra)
	}

	buf := new(bytes.Buffer)
	buf.Write(make([]byte, 16))
	name := [4]byte{'B', 'K', 'Q', '1'}
	if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(88)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := [6]byte{'6', '0', '0', '0', '0', '0'}
	if err := binary.Write(buf, binary.LittleEndian, code); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 16))
	itemName := make([]byte, 24)
	copy(itemName, "BANK")
	buf.Write(itemName)
	buf.Write(make([]byte, 20))
	for i := 1; i <= 18; i++ {
		writeMACFloat32(t, buf, float32(i))
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(321)); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 2))
	for i := 19; i <= 31; i++ {
		writeMACFloat32(t, buf, float32(i))
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.Stocks) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.Stocks[0]
	if item.Symbol != "600000" || item.Name != "BANK" {
		t.Fatalf("unexpected item: %+v", item)
	}
	if item.Close != 5 || item.CurrentVol != 321 || item.PEStatic != 29 || item.PETTM != 30 {
		t.Fatalf("unexpected metrics: %+v", item)
	}
}

func TestMACBoardMembersQuotesDynamicBuildRequestAndParseResponse(t *testing.T) {
	boardCode, err := ExchangeMACBoardCode("880761")
	if err != nil {
		t.Fatal(err)
	}
	bitmap := [20]byte{0x31} // bits: 0=pre_close 4=close 5=vol
	msg := NewMACBoardMembersQuotesDynamic(&MACBoardMembersQuotesDynamicRequest{
		BoardCode:   boardCode,
		SortType:    14,
		Start:       0,
		PageSize:    10,
		SortOrder:   1,
		FieldBitmap: bitmap,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACBOARDMEMBERS {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACBoardMembersQuotesDynamicRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.BoardCode != boardCode || req.PageSize != 10 || req.SortOrder != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}
	if req.FieldBitmap != bitmap {
		t.Fatalf("unexpected field bitmap: %#v", req.FieldBitmap)
	}

	buf := new(bytes.Buffer)
	buf.Write(bitmap[:])
	if err := binary.Write(buf, binary.LittleEndian, uint32(88)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}

	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := make([]byte, 22)
	copy(code, "600000")
	buf.Write(code)
	name := make([]byte, 44)
	copy(name, "BANK")
	buf.Write(name)
	writeMACFloat32(t, buf, 10.1)
	writeMACFloat32(t, buf, 10.5)
	if err := binary.Write(buf, binary.LittleEndian, uint32(1234)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Total != 88 || reply.Count != 1 || len(reply.ActiveFields) != 3 || len(reply.Stocks) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.ActiveFields[0].Name != "pre_close" || reply.ActiveFields[1].Name != "close" || reply.ActiveFields[2].Name != "vol" {
		t.Fatalf("unexpected active fields: %+v", reply.ActiveFields)
	}
	item := reply.Stocks[0]
	if item.Symbol != "600000" || item.Name != "BANK" {
		t.Fatalf("unexpected item header: %+v", item)
	}
	assertNearFloat64(t, item.Values["pre_close"].(float64), 10.1, "dynamic pre_close")
	assertNearFloat64(t, item.Values["close"].(float64), 10.5, "dynamic close")
	if item.Values["vol"].(uint32) != 1234 {
		t.Fatalf("unexpected dynamic vol: %+v", item.Values)
	}
}

func TestMACQuotesBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACQuotes(&MACQuotesRequest{
		Market: 1,
		Code:   makeMACCode22("600000"),
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACQUOTES {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACQuotesRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Code != makeMACCode22("600000") || req.One != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, makeMACCode22("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260418)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint8(7)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.5)
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}

	if err := binary.Write(buf, binary.LittleEndian, uint16(570)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.1)
	writeMACFloat32(t, buf, 10.0)
	if err := binary.Write(buf, binary.LittleEndian, uint32(1234)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 0.5)

	if err := binary.Write(buf, binary.LittleEndian, uint16(571)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.2)
	writeMACFloat32(t, buf, 10.1)
	if err := binary.Write(buf, binary.LittleEndian, uint32(2234)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 0.8)

	name := make([]byte, 44)
	copy(name, "PingAn Bank")
	buf.Write(name)
	if err := binary.Write(buf, binary.LittleEndian, uint8(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(6)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 100)
	buf.Write(make([]byte, 5))
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260418)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(93005)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 9.9)
	writeMACFloat32(t, buf, 10.0)
	writeMACFloat32(t, buf, 10.8)
	writeMACFloat32(t, buf, 9.8)
	writeMACFloat32(t, buf, 10.5)
	writeMACFloat32(t, buf, 1.1)
	if err := binary.Write(buf, binary.LittleEndian, uint32(9988)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 123456.5)
	buf.Write(make([]byte, 12))
	writeMACFloat32(t, buf, 2.5)
	writeMACFloat32(t, buf, 10.2)
	if err := binary.Write(buf, binary.LittleEndian, uint32(42)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Code != "600000" || reply.Name != "PingAn Bank" || reply.Count != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if len(reply.ChartData) != 2 || reply.ChartData[0].Time != "09:30:00" || reply.ChartData[1].Vol != 2234 {
		t.Fatalf("unexpected chart data: %+v", reply.ChartData)
	}
	assertNearFloat64(t, reply.Close, 10.5, "close")
	assertNearFloat64(t, reply.Turnover, 2.5, "turnover")
	assertNearFloat64(t, reply.Avg, 10.2, "avg")
	if reply.Industry != 42 {
		t.Fatalf("unexpected summary: %+v", reply)
	}
}

func TestMACSymbolBelongBoardBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACSymbolBelongBoard(&MACSymbolBelongBoardRequest{
		Market: 1,
		Symbol: makeMACCode8("600000"),
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACSYMBOLBELONGBOARD {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACSymbolBelongBoardRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || string(req.Symbol[:6]) != "600000" || string(req.Query[:10]) != "Stock_GLHQ" {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	query := make([]byte, 12)
	copy(query, "Stock_GLHQ")
	buf.Write(query)
	buf.Write(make([]byte, 13))
	buf.WriteString(`[["HY",1,"880001","Coal",10.5,10.0,1,2,3]]`)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Market != 1 || reply.Query != "Stock_GLHQ" || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.List[0].BoardCode != "880001" || reply.List[0].BoardName != "Coal" {
		t.Fatalf("unexpected belong board item: %+v", reply.List[0])
	}
}

func TestMACSymbolBarsBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACSymbolBars(&MACSymbolBarsRequest{
		Market: 1,
		Code:   makeMACCode22("600000"),
		Period: KLINE_TYPE_DAILY,
		Count:  1,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACSYMBOLBARS {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACSymbolBarsRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Period != KLINE_TYPE_DAILY || req.Count != 1 || req.Flag1 != 1 || req.Flag2 != 1 || req.Flag4 != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := make([]byte, 12)
	copy(code, "600000")
	buf.Write(code)
	buf.Write(make([]byte, 10))
	if err := buf.WriteByte(byte(KLINE_TYPE_DAILY)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260331)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(34200)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.1)
	writeMACFloat32(t, buf, 10.8)
	writeMACFloat32(t, buf, 9.9)
	writeMACFloat32(t, buf, 10.5)
	writeMACFloat32(t, buf, 12345.6)
	writeMACFloat32(t, buf, 789.0)
	writeMACFloat32(t, buf, 456.0)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.DateTime != "2026-03-31 09:30:00" || math.Abs(item.Close-10.5) > 0.001 || math.Abs(item.Vol-789.0) > 0.001 {
		t.Fatalf("unexpected symbol bar: %+v", item)
	}
}

func TestCombineMACDateTimeOvernight(t *testing.T) {
	got := combineMACDateTime(20260331, 60, true).Format("2006-01-02 15:04:05")
	if got != "2026-04-01 00:01:00" {
		t.Fatalf("unexpected datetime: %s", got)
	}
}

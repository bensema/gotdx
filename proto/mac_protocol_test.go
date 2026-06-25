package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
	"time"
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

func TestMACServerInfoBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACServerInfo(nil)

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACSERVERINFO {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}
	if len(raw[12:]) != 68 || raw[12] != 0x04 || raw[15] != 0x31 || raw[24] != 0x00 || raw[25] != 0x27 {
		t.Fatalf("unexpected mac server info payload: %x", raw[12:])
	}

	buf := new(bytes.Buffer)
	must := func(v interface{}) {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	must(uint16(1))
	buf.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	buf.Write([]byte{'-', '1', 0})
	buf.Write(make([]byte, 9))
	must(uint32(20260516))
	must(uint32(93000))
	for _, value := range []uint16{570, 690, 780, 900, 0, 0, 0, 0} {
		must(value)
	}
	for _, value := range []uint16{540, 660, 1260, 1380, 0, 0, 0, 0} {
		must(value)
	}
	buf.WriteByte(7)
	must(uint32(20260515))
	must(uint32(1))
	must(uint32(20260514))
	must(uint32(2))
	must(uint32(10))
	must(uint32(20))
	buf.Write([]byte{0xaa, 0xbb})

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || reply.FlagsHex != "0102030405060708" || reply.Tag != "-1" || reply.Today != "2026-05-16" {
		t.Fatalf("unexpected reply header: %+v", reply)
	}
	if len(reply.Sessions1) != 4 || reply.Sessions1[0].Open != "9:30" || reply.Sessions1[1].Close != "15:00" {
		t.Fatalf("unexpected sessions1: %+v", reply.Sessions1)
	}
	if reply.LastTradingDay != "2026-05-15" || reply.MarketParam1 != 10 || reply.MarketParam2 != 20 || reply.ExtraHex != "aabb" {
		t.Fatalf("unexpected reply tail: %+v", reply)
	}
}

func TestMACKLineOffsetBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACKLineOffset(&MACKLineOffsetRequest{Offset: 3, Count: 128000})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACKLINEOFFSET {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}
	var req MACKLineOffsetRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Offset != 3 || req.Count != 128000 {
		t.Fatalf("unexpected request: %+v", req)
	}

	payload := []byte{0x00, 0x01, 0xf4, 0x00, 0x02, 0x00, 0x00, 0x00}
	if err := msg.ParseResponse(&RespHeader{}, payload); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Total != 128000 || reply.Returned != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
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
	writeMACFloat32(t, buf, 1) // pre_close
	writeMACFloat32(t, buf, 2) // open
	writeMACFloat32(t, buf, 3) // high
	writeMACFloat32(t, buf, 4) // low
	writeMACFloat32(t, buf, 5) // close
	if err := binary.Write(buf, binary.LittleEndian, uint32(600)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 7)  // vol_ratio
	writeMACFloat32(t, buf, 8)  // amount
	writeMACFloat32(t, buf, 9)  // total_shares
	writeMACFloat32(t, buf, 10) // float_shares
	writeMACFloat32(t, buf, 11) // eps
	writeMACFloat32(t, buf, 12) // net_assets
	writeMACFloat32(t, buf, 13) // action_price
	writeMACFloat32(t, buf, 14) // market_cap_ab
	writeMACFloat32(t, buf, 15) // pe_dynamic
	if err := binary.Write(buf, binary.LittleEndian, uint32(1600)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 17) // unknown_23
	writeMACFloat32(t, buf, 18) // dividend_yield
	if err := binary.Write(buf, binary.LittleEndian, uint32(321)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 20) // turnover
	if err := binary.Write(buf, binary.LittleEndian, uint32(21)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(22)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 23) // buy_price_limit
	writeMACFloat32(t, buf, 24) // sell_price_limit
	if err := binary.Write(buf, binary.LittleEndian, uint32(25)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(26)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 27) // pre_ipov
	writeMACFloat32(t, buf, 28) // speed_pct
	if err := binary.Write(buf, binary.LittleEndian, uint32(29)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 30) // pe_ttm
	writeMACFloat32(t, buf, 31) // pe_static
	writeMACFloat32(t, buf, 32) // unknown_close_price

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
	if item.Close != 5 || item.Vol != 600 || item.LastVolume != 321 || item.CurrentVol != 321 {
		t.Fatalf("unexpected volume fields: %+v", item)
	}
	if item.SpeedPct != 28 || item.RiseSpeed != 28 || item.PEStatic != 31 || item.PETTM != 30 {
		t.Fatalf("unexpected corrected metrics: %+v", item)
	}
	if item.NetAssets != 12 || item.ActionPrice != 13 || item.Unknown23 != 17 || item.Turnover != 20 {
		t.Fatalf("unexpected aligned aliases: %+v", item)
	}
	if item.DividendYield != 18 || item.FlagKCB != 29 || item.KCBFlag != 29 || item.LotSizeBoardSymbol != "880226" {
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

func TestMACBoardMembersQuotesDynamicSupportsSignedAndAliasFields(t *testing.T) {
	bitmap := [20]byte{}
	bitmap[0x24/8] |= 1 << (0x24 % 8)
	bitmap[0x58/8] |= 1 << (0x58 % 8)
	bitmap[0x8e/8] |= 1 << (0x8e % 8)

	msg := NewMACBoardMembersQuotesDynamic(&MACBoardMembersQuotesDynamicRequest{FieldBitmap: bitmap})
	buf := new(bytes.Buffer)
	buf.Write(bitmap[:])
	if err := binary.Write(buf, binary.LittleEndian, uint32(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := make([]byte, 22)
	copy(code, "000001")
	buf.Write(code)
	name := make([]byte, 44)
	copy(name, "PINGAN")
	buf.Write(name)
	writeMACFloat32(t, buf, 12.3) // pre_ipov
	if err := binary.Write(buf, binary.LittleEndian, int32(-7)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(-1)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.ActiveFields[0].Name != "pre_ipov" || reply.ActiveFields[0].Aliases[0] != "float_shares" {
		t.Fatalf("unexpected alias field: %+v", reply.ActiveFields)
	}
	item := reply.Stocks[0]
	assertNearFloat64(t, item.Values["pre_ipov"].(float64), 12.3, "dynamic pre_ipov")
	assertNearFloat64(t, item.Values["float_shares"].(float64), 12.3, "dynamic alias float_shares")
	if item.Values["annual_limit_up_days"].(int32) != -7 || item.Values["constant_neg_one"].(int32) != -1 {
		t.Fatalf("unexpected signed dynamic values: %+v", item.Values)
	}
}

func TestMACDynamicFieldMapAlignsLatestTDX(t *testing.T) {
	bitmap := [20]byte{}
	bits := []int{0x16, 0x37, 0x3e, 0x48, 0x73, 0x85, 0x8c, 0x8f}
	for _, bit := range bits {
		bitmap[bit/8] |= 1 << (bit % 8)
	}

	fields := activeMACDynamicFields(bitmap)
	want := map[uint8]struct {
		name   string
		format string
		alias  string
	}{
		0x16: {name: "board_strength", format: "float32", alias: "unknown_22"},
		0x37: {name: "index_metric", format: "float32", alias: "unknown_55"},
		0x3e: {name: "stock_class_code", format: "uint32", alias: "unknown_62"},
		0x48: {name: "bid2_price", format: "float32", alias: "low_copy"},
		0x73: {name: "ddx", format: "float32"},
		0x85: {name: "ask5_price", format: "float32", alias: "avg_price_copy"},
		0x8c: {name: "bid_ask_diff", format: "int32"},
		0x8f: {name: "stock_rating", format: "float32"},
	}

	if len(fields) != len(bits) {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	for _, field := range fields {
		expect, ok := want[field.Bit]
		if !ok {
			t.Fatalf("unexpected field bit: %+v", field)
		}
		if field.Name != expect.name || field.Format != expect.format {
			t.Fatalf("unexpected field def for bit %#x: %+v", field.Bit, field)
		}
		if expect.alias != "" && !containsString(field.Aliases, expect.alias) {
			t.Fatalf("missing alias %q for field %+v", expect.alias, field)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
	if err := binary.Write(buf, binary.LittleEndian, uint32(83005)); err != nil {
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
	if reply.Industry != 83005 || reply.IndustryCode != "881282" {
		t.Fatalf("unexpected summary: %+v", reply)
	}
}

func TestMACQuotesBuildRequestWithDate(t *testing.T) {
	queryDate := uint32(20260418)
	msg := NewMACQuotes(&MACQuotesRequest{
		Market: 1,
		Code:   makeMACCode22("600000"),
		Zero1:  uint16(queryDate & 0xffff),
		Zero2:  uint16(queryDate >> 16),
	})

	raw := mustBuildRequest(t, msg)
	var req MACQuotesRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	gotDate := uint32(req.Zero1) | (uint32(req.Zero2) << 16)
	if gotDate != queryDate || req.One != 1 {
		t.Fatalf("unexpected request date: got=%d req=%+v", gotDate, req)
	}
}

func TestMACSymbolQuotesBuildRequestAndParseResponse(t *testing.T) {
	var bitmap [20]byte
	for _, bit := range []int{0x0, 0x4, 0x5, 0x4a} {
		bitmap[bit/8] |= 1 << uint(bit%8)
	}
	msg := NewMACSymbolQuotes(&MACSymbolQuotesRequest{
		FieldBitmap: bitmap,
		Stocks: []MACSymbolQuoteStock{
			{Market: 0, Code: makeMACCode22("000001")},
			{Market: 1, Code: makeMACCode22("600000")},
		},
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACSYMBOLQUOTES {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}
	if !bytes.Equal(raw[12:32], bitmap[:]) {
		t.Fatalf("unexpected field bitmap: got=%x want=%x", raw[12:32], bitmap)
	}
	if got := binary.LittleEndian.Uint16(raw[32:34]); got != 2 {
		t.Fatalf("unexpected stock count: %d", got)
	}
	if got := binary.LittleEndian.Uint16(raw[34:36]); got != 0 {
		t.Fatalf("unexpected first market: %d", got)
	}
	if got := Utf8ToGbk(raw[36:58]); got != "000001" {
		t.Fatalf("unexpected first code: %q", got)
	}
	if got := binary.LittleEndian.Uint16(raw[58:60]); got != 1 {
		t.Fatalf("unexpected second market: %d", got)
	}
	if got := Utf8ToGbk(raw[60:82]); got != "600000" {
		t.Fatalf("unexpected second code: %q", got)
	}

	buf := new(bytes.Buffer)
	buf.Write(bitmap[:])
	if err := binary.Write(buf, binary.LittleEndian, uint32(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		market    uint16
		symbol    string
		name      string
		preClose  float32
		close     float32
		vol       uint32
		ahCodeRaw uint32
	}{
		{market: 0, symbol: "000001", name: "PingAn Bank", preClose: 10.1, close: 10.5, vol: 123456, ahCodeRaw: 700},
		{market: 1, symbol: "600000", name: "PuFa Bank", preClose: 11.1, close: 11.8, vol: 654321, ahCodeRaw: 6881},
	} {
		if err := binary.Write(buf, binary.LittleEndian, item.market); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, makeMACCode22(item.symbol)); err != nil {
			t.Fatal(err)
		}
		name := make([]byte, 44)
		copy(name, item.name)
		buf.Write(name)
		writeMACFloat32(t, buf, item.preClose)
		writeMACFloat32(t, buf, item.close)
		if err := binary.Write(buf, binary.LittleEndian, item.vol); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, item.ahCodeRaw); err != nil {
			t.Fatal(err)
		}
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Total != 2 || reply.Count != 2 || len(reply.ActiveFields) != 4 || len(reply.Stocks) != 2 {
		t.Fatalf("unexpected reply counts: %+v", reply)
	}
	if reply.ActiveFields[3].Name != "ah_code" {
		t.Fatalf("unexpected active fields: %+v", reply.ActiveFields)
	}
	first := reply.Stocks[0]
	second := reply.Stocks[1]
	if first.Symbol != "000001" || first.Name != "PingAn Bank" || second.Symbol != "600000" {
		t.Fatalf("unexpected symbols: %+v %+v", first, second)
	}
	assertNearFloat64(t, first.Values["pre_close"].(float64), 10.1, "symbol_quotes_pre_close_0")
	assertNearFloat64(t, second.Values["close"].(float64), 11.8, "symbol_quotes_close_1")
	if first.Values["vol"].(uint32) != 123456 || second.Values["ah_code"].(uint32) != 6881 {
		t.Fatalf("unexpected symbol quote values: %+v %+v", first.Values, second.Values)
	}
}

func TestMACMarketMonitorBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACMarketMonitor(&MACMarketMonitorRequest{
		Market: 1,
		Start:  5,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACMARKETMONITOR {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACMarketMonitorRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	wantLimits := [5]uint16{200, 30, 40, 50, 200}
	if req.Market != 1 || req.Start != 5 || req.Count != 600 || req.Mode != 1 || req.Limits != wantLimits {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}

	item := make([]byte, 32)
	binary.LittleEndian.PutUint16(item[0:2], 1)
	copy(item[2:8], "600000")
	item[9] = 0x0b
	binary.LittleEndian.PutUint16(item[11:13], 321)
	item[15] = 1
	binary.LittleEndian.PutUint32(item[16:20], math.Float32bits(2.5))
	binary.LittleEndian.PutUint32(item[20:24], math.Float32bits(0.032))
	binary.LittleEndian.PutUint32(item[24:28], math.Float32bits(7.7))
	item[29] = 14
	binary.LittleEndian.PutUint16(item[30:32], 3015)
	buf.Write(item)
	buf.WriteString("PingAn,")

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	got := reply.List[0]
	if got.Index != 321 || got.Market != 1 || got.Code != "600000" || got.Name != "PingAn" {
		t.Fatalf("unexpected item identity: %+v", got)
	}
	if got.Time != "14:30:15" || got.Desc != "区间放量涨" || got.Value != "2.5倍3.20%" || got.UnusualType != 0x0b || got.V1 != 1 {
		t.Fatalf("unexpected item text fields: %+v", got)
	}
	assertNearFloat64(t, got.V2, 2.5, "v2")
	assertNearFloat64(t, got.V3, 0.032, "v3")
	assertNearFloat64(t, got.V4, 7.7, "v4")
}

func TestMACTransactionsBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACTransactions(&MACTransactionsRequest{
		Market:    1,
		Code:      makeMACCode22("600000"),
		QueryDate: 20260418,
		Start:     5,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACTRANSACTIONS {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACTransactionsRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Code != makeMACCode22("600000") || req.QueryDate != 20260418 || req.Start != 5 || req.Count != 1000 {
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
	if err := buf.WriteByte(0); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(34215)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.5)
	if err := binary.Write(buf, binary.LittleEndian, uint32(100)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(55800)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.8)
	if err := binary.Write(buf, binary.LittleEndian, uint32(80)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Market != 1 || reply.Code != "600000" || reply.QueryDate != 20260418 || reply.Count != 2 || reply.Start != 5 || reply.Total != 20 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if len(reply.List) != 2 || reply.List[0].Time != "09:30:15" || reply.List[1].BuyOrSell != 1 {
		t.Fatalf("unexpected list: %+v", reply.List)
	}
	assertNearFloat64(t, reply.List[0].Price, 10.5, "transaction_price_0")
	assertNearFloat64(t, reply.List[1].Price, 10.8, "transaction_price_1")
}

func TestMACAuctionBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACAuction(&MACAuctionRequest{
		Market: 1,
		Code:   makeMACCode22("600000"),
		Start:  3,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACAUCTION {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACAuctionRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Code != makeMACCode22("600000") || req.Start != 3 || req.Count != 500 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, makeMACCode22("600000")); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(2)); err != nil {
		t.Fatal(err)
	}
	buf.Write(make([]byte, 8))
	if err := binary.Write(buf, binary.LittleEndian, uint32(34215)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.5)
	if err := binary.Write(buf, binary.LittleEndian, uint32(100)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(-50)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(55800)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.8)
	if err := binary.Write(buf, binary.LittleEndian, uint32(120)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, int32(30)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Market != 1 || reply.Code != "600000" || reply.Count != 2 || len(reply.List) != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.List[0].Time != "09:30:15" || reply.List[0].Unmatched != -50 || reply.List[0].Flag != -1 {
		t.Fatalf("unexpected first auction item: %+v", reply.List[0])
	}
	if reply.List[1].Time != "15:30:00" || reply.List[1].Unmatched != 30 || reply.List[1].Flag != 1 {
		t.Fatalf("unexpected second auction item: %+v", reply.List[1])
	}
	assertNearFloat64(t, reply.List[0].Price, 10.5, "auction_price_0")
	assertNearFloat64(t, reply.List[1].Price, 10.8, "auction_price_1")
}

func TestMACTickChartsBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACTickCharts(&MACTickChartsRequest{
		Market:    1,
		Code:      makeMACCode22("600000"),
		QueryDate: 20260418,
		Days:      2,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACTICKCHARTS {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACTickChartsRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Code != makeMACCode22("600000") || req.QueryDate != 20260418 || req.Days != 2 || req.One != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, makeMACCode22("600000")); err != nil {
		t.Fatal(err)
	}
	for _, value := range []uint32{20260418, 20260417, 0, 0, 0} {
		if err := binary.Write(buf, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []float32{10.0, 9.8, 0, 0, 0} {
		writeMACFloat32(t, buf, value)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(1); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(4)); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		minutes uint16
		price   float32
		avg     float32
		vol     uint16
		unknown uint16
	}{
		{minutes: 570, price: 10.1, avg: 10.05, vol: 100, unknown: 7},
		{minutes: 571, price: 10.2, avg: 10.10, vol: 120, unknown: 8},
		{minutes: 570, price: 9.9, avg: 9.85, vol: 90, unknown: 9},
		{minutes: 571, price: 10.0, avg: 9.90, vol: 110, unknown: 10},
	} {
		if err := binary.Write(buf, binary.LittleEndian, item.minutes); err != nil {
			t.Fatal(err)
		}
		writeMACFloat32(t, buf, item.price)
		writeMACFloat32(t, buf, item.avg)
		if err := binary.Write(buf, binary.LittleEndian, item.vol); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, item.unknown); err != nil {
			t.Fatal(err)
		}
	}
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
	if err := binary.Write(buf, binary.LittleEndian, uint32(150005)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.0)
	writeMACFloat32(t, buf, 10.1)
	writeMACFloat32(t, buf, 10.3)
	writeMACFloat32(t, buf, 9.9)
	writeMACFloat32(t, buf, 10.2)
	writeMACFloat32(t, buf, 0.2)
	if err := binary.Write(buf, binary.LittleEndian, uint32(220)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 12345.6)
	buf.Write(make([]byte, 12))
	writeMACFloat32(t, buf, 2.5)
	writeMACFloat32(t, buf, 10.1)
	if err := binary.Write(buf, binary.LittleEndian, uint32(83005)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Market != 1 || reply.Code != "600000" || reply.Count != 2 || reply.PageSize != 2 || len(reply.Charts) != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.Charts[0].Date != "2026-04-18" || len(reply.Charts[0].Ticks) != 2 || reply.Charts[1].Ticks[1].Unknown != 10 {
		t.Fatalf("unexpected charts: %+v", reply.Charts)
	}
	if reply.Name != "PingAn Bank" || reply.DateTime.Format(time.DateTime) != "2026-04-18 15:00:05" || reply.Industry != 83005 || reply.IndustryCode != "881282" {
		t.Fatalf("unexpected summary: %+v", reply)
	}
	assertNearFloat64(t, reply.Charts[0].PreClose, 10.0, "tick_day_pre_close_0")
	assertNearFloat64(t, reply.Charts[1].PreClose, 9.8, "tick_day_pre_close_1")
	assertNearFloat64(t, reply.Close, 10.2, "tick_close")
}

func TestMACTickChartsParseResponseSupportsPartialLeadingDay(t *testing.T) {
	msg := NewMACTickCharts(nil)

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, makeMACCode22("600000")); err != nil {
		t.Fatal(err)
	}
	for _, value := range []uint32{20260506, 20260430, 20260429, 0, 0} {
		if err := binary.Write(buf, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []float32{10.0, 9.8, 9.6, 0, 0} {
		writeMACFloat32(t, buf, value)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(3)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(1); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(4)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(9)); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		minutes uint16
		price   float32
		avg     float32
		vol     uint16
		unknown uint16
	}{
		{minutes: 570, price: 10.1, avg: 10.1, vol: 1, unknown: 1},
		{minutes: 570, price: 9.9, avg: 9.9, vol: 2, unknown: 2},
		{minutes: 571, price: 10.0, avg: 9.95, vol: 3, unknown: 3},
		{minutes: 572, price: 10.1, avg: 10.00, vol: 4, unknown: 4},
		{minutes: 573, price: 10.2, avg: 10.05, vol: 5, unknown: 5},
		{minutes: 570, price: 9.7, avg: 9.7, vol: 6, unknown: 6},
		{minutes: 571, price: 9.8, avg: 9.75, vol: 7, unknown: 7},
		{minutes: 572, price: 9.9, avg: 9.80, vol: 8, unknown: 8},
		{minutes: 573, price: 10.0, avg: 9.85, vol: 9, unknown: 9},
	} {
		if err := binary.Write(buf, binary.LittleEndian, item.minutes); err != nil {
			t.Fatal(err)
		}
		writeMACFloat32(t, buf, item.price)
		writeMACFloat32(t, buf, item.avg)
		if err := binary.Write(buf, binary.LittleEndian, item.vol); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, item.unknown); err != nil {
			t.Fatal(err)
		}
	}
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
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260506)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(93000)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.0)
	writeMACFloat32(t, buf, 10.1)
	writeMACFloat32(t, buf, 10.3)
	writeMACFloat32(t, buf, 9.9)
	writeMACFloat32(t, buf, 10.2)
	writeMACFloat32(t, buf, 0.2)
	if err := binary.Write(buf, binary.LittleEndian, uint32(220)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 12345.6)
	buf.Write(make([]byte, 12))
	writeMACFloat32(t, buf, 2.5)
	writeMACFloat32(t, buf, 10.1)
	if err := binary.Write(buf, binary.LittleEndian, uint32(83005)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Total != 9 || reply.PageSize != 4 || len(reply.Charts) != 3 {
		t.Fatalf("unexpected reply counts: %+v", reply)
	}
	if len(reply.Charts[0].Ticks) != 1 || len(reply.Charts[1].Ticks) != 4 || len(reply.Charts[2].Ticks) != 4 {
		t.Fatalf("unexpected chart split: %+v", reply.Charts)
	}
	if reply.Charts[0].Date != "2026-05-06" || reply.Charts[1].Date != "2026-04-30" || reply.Charts[2].Date != "2026-04-29" {
		t.Fatalf("unexpected chart dates: %+v", reply.Charts)
	}
	if reply.Charts[1].Ticks[0].Time != "09:30:00" || reply.Charts[2].Ticks[3].Unknown != 9 {
		t.Fatalf("unexpected tick data: %+v", reply.Charts)
	}
}

func TestMACSymbolInfoBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACSymbolInfo(&MACSymbolInfoRequest{
		Market: 1,
		Code:   makeMACCode22("600000"),
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACSYMBOLINFO {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACSymbolInfoRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Code != makeMACCode22("600000") || req.One != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	buf.Write(make([]byte, 8))
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, makeMACCode22("600000")); err != nil {
		t.Fatal(err)
	}
	name := make([]byte, 44)
	copy(name, "PingAn Bank")
	buf.Write(name)
	buf.Write(make([]byte, 20))
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260418)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(150005)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(321)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 10.0)
	writeMACFloat32(t, buf, 10.1)
	writeMACFloat32(t, buf, 10.3)
	writeMACFloat32(t, buf, 9.9)
	writeMACFloat32(t, buf, 10.2)
	writeMACFloat32(t, buf, 0.2)
	if err := binary.Write(buf, binary.LittleEndian, uint32(220)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 12345.6)
	if err := binary.Write(buf, binary.LittleEndian, uint32(100)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(120)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(11)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 22.5)
	buf.Write(make([]byte, 20))
	if err := binary.Write(buf, binary.LittleEndian, uint32(33)); err != nil {
		t.Fatal(err)
	}
	writeMACFloat32(t, buf, 1.5)
	writeMACFloat32(t, buf, 2.5)
	writeMACFloat32(t, buf, 10.1)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Market != 1 || reply.Code != "600000" || reply.Name != "PingAn Bank" || reply.DateTime.Format(time.DateTime) != "2026-04-18 15:00:05" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.Activity != 321 || reply.Vol != 220 || reply.InsideVolume != 100 || reply.OutsideVolume != 120 || reply.Decimal != 2 || reply.UnknownA != 11 || reply.UnknownC != 33 {
		t.Fatalf("unexpected metrics: %+v", reply)
	}
	assertNearFloat64(t, reply.Close, 10.2, "symbol_info_close")
	assertNearFloat64(t, reply.Amount, 12345.6, "symbol_info_amount")
	assertNearFloat64(t, reply.UnknownB, 22.5, "symbol_info_unknown_b")
	assertNearFloat64(t, reply.VR, 1.5, "symbol_info_vr")
	assertNearFloat64(t, reply.Turnover, 2.5, "symbol_info_turnover")
	assertNearFloat64(t, reply.Avg, 10.1, "symbol_info_avg")
}

func TestMACCapitalFlowBuildRequestAndParseResponse(t *testing.T) {
	msg := NewMACCapitalFlow(&MACCapitalFlowRequest{
		Market: 1,
		Symbol: makeMACCode8("000001"),
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x02 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACCAPITALFLOW {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req MACCapitalFlowRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Symbol != makeMACCode8("000001") || req.Query != makeMACCode21("Stock_ZJLX") {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	queryInfo := make([]byte, 12)
	copy(queryInfo, "Stock_ZJLX")
	buf.Write(queryInfo)
	buf.Write(make([]byte, 5))
	ext := make([]byte, 8)
	copy(ext, "000001")
	buf.Write(ext)
	buf.WriteString(`[[100,80,60,40],[500,400,30,20,10,5]]`)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Market != 1 || reply.QueryInfo != "Stock_ZJLX" || reply.Ext != "000001" {
		t.Fatalf("unexpected reply header: %+v", reply)
	}
	if len(reply.Today) != 4 || len(reply.FiveDays) != 6 {
		t.Fatalf("unexpected raw lists: %+v", reply)
	}
	assertNearFloat64(t, reply.TodayMainIn, 100, "capital_flow_today_main_in")
	assertNearFloat64(t, reply.TodayMainNetIn, 20, "capital_flow_today_main_net")
	assertNearFloat64(t, reply.TodayRetailNetIn, 20, "capital_flow_today_retail_net")
	assertNearFloat64(t, reply.FiveDayMainNetIn, 100, "capital_flow_five_day_main_net")
	assertNearFloat64(t, reply.FiveDaySuperNet, 30, "capital_flow_five_day_super_net")
}

func TestMACFileListBuildRequestAndParseResponse(t *testing.T) {
	req := MACFileListRequest{Offset: 16}
	copy(req.Filename[:], "StockInfo.dat")
	msg := NewMACFileList(&req)

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACFILELIST {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var gotReq MACFileListRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &gotReq); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if gotReq.Offset != 16 || string(gotReq.Filename[:13]) != "StockInfo.dat" {
		t.Fatalf("unexpected request: %+v", gotReq)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(16)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(1024)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(byte(2)); err != nil {
		t.Fatal(err)
	}
	hash := make([]byte, 32)
	copy(hash, "abcdef1234567890")
	buf.Write(hash)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Offset != 16 || reply.Size != 1024 || reply.Flag != 2 || reply.Hash != "abcdef1234567890" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestMACFileDownloadBuildRequestAndParseResponse(t *testing.T) {
	req := MACFileDownloadRequest{Index: 3, Offset: 64, Size: 128}
	copy(req.Filename[:], "StockInfo.dat")
	msg := NewMACFileDownload(&req)

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_MACFILEDOWNLOAD {
		t.Fatalf("unexpected request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var gotReq MACFileDownloadRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &gotReq); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if gotReq.Index != 3 || gotReq.Offset != 64 || gotReq.Size != 128 || string(gotReq.Filename[:13]) != "StockInfo.dat" {
		t.Fatalf("unexpected request: %+v", gotReq)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(3)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(5)); err != nil {
		t.Fatal(err)
	}
	buf.WriteString("hello")

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Index != 3 || reply.Size != 5 || string(reply.Data) != "hello" {
		t.Fatalf("unexpected reply: %+v", reply)
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
	if reply.List[0].MarketCode != 1 || reply.List[0].LimitUpCount != 1 || reply.List[0].LimitDownCount != 2 || reply.List[0].MostSimilar != 3 || reply.List[0].SchemaColumns != 9 {
		t.Fatalf("unexpected belong board metrics: %+v", reply.List[0])
	}
}

func TestMACSymbolBelongBoardParseResponseWithExpandedSchema(t *testing.T) {
	msg := NewMACSymbolBelongBoard(&MACSymbolBelongBoardRequest{
		Market: 1,
		Symbol: makeMACCode8("600000"),
	})

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	query := make([]byte, 12)
	copy(query, "Stock_GLHQ")
	buf.Write(query)
	buf.Write(make([]byte, 13))
	buf.WriteString(`[["HY",1,"880001","Coal",10.5,10.0,2.5,1,"600001","Peer",11.1,10.9,1.2]]`)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.SchemaColumns != 13 || item.SpeedPct != 2.5 || item.SymbolMarket != 1 || item.Symbol != "600001" || item.SymbolName != "Peer" {
		t.Fatalf("unexpected expanded belong board item: %+v", item)
	}
	if item.SymbolClose != 11.1 || item.SymbolPreClose != 10.9 || item.SymbolSpeedPct != 1.2 {
		t.Fatalf("unexpected expanded belong board peer metrics: %+v", item)
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
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260331)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(150005)); err != nil {
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
	if err := binary.Write(buf, binary.LittleEndian, uint32(83005)); err != nil {
		t.Fatal(err)
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.DateTime.Format(time.DateTime) != "2026-03-31 09:30:00" || math.Abs(item.Close-10.5) > 0.001 || math.Abs(item.Vol-789.0) > 0.001 {
		t.Fatalf("unexpected symbol bar: %+v", item)
	}
	if reply.Name != "PingAn Bank" || reply.DateTime.Format(time.DateTime) != "2026-03-31 15:00:05" || reply.Industry != 83005 || reply.IndustryCode != "881282" {
		t.Fatalf("unexpected symbol bar summary: %+v", reply)
	}
}

func TestCombineMACDateTimeOvernight(t *testing.T) {
	got := combineMACDateTime(20260331, 60, true).Format("2006-01-02 15:04:05")
	if got != "2026-04-01 00:01:00" {
		t.Fatalf("unexpected datetime: %s", got)
	}
}

package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func readExReqHeader(t *testing.T, raw []byte) exReqHeader {
	t.Helper()
	var header exReqHeader
	if err := binary.Read(bytes.NewReader(raw[:10]), binary.LittleEndian, &header); err != nil {
		t.Fatalf("read ex header failed: %v", err)
	}
	return header
}

func writeFloat32(t *testing.T, buf *bytes.Buffer, value float32) {
	t.Helper()
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(value)); err != nil {
		t.Fatalf("write float32 failed: %v", err)
	}
}

func buildExQuoteRecord(t *testing.T, codeLen int, category byte, code string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)

	if err := buf.WriteByte(category); err != nil {
		t.Fatalf("write category failed: %v", err)
	}
	codeBuf := make([]byte, codeLen)
	copy(codeBuf, code)
	if _, err := buf.Write(codeBuf); err != nil {
		t.Fatalf("write code failed: %v", err)
	}

	must := func(v interface{}) {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			t.Fatalf("binary write failed: %v", err)
		}
	}

	must(uint32(123))
	writeFloat32(t, buf, 10.1)
	writeFloat32(t, buf, 10.2)
	writeFloat32(t, buf, 10.5)
	writeFloat32(t, buf, 9.9)
	writeFloat32(t, buf, 10.3)
	must(uint32(11))
	must(uint32(12))
	must(uint32(13))
	must(uint32(14))
	writeFloat32(t, buf, 15.5)
	must(uint32(16))
	must(uint32(17))
	must(uint32(18))
	must(uint32(19))

	for i := 0; i < 5; i++ {
		writeFloat32(t, buf, float32(i+1))
	}
	for i := 0; i < 5; i++ {
		must(uint32(101 + i))
	}
	for i := 0; i < 5; i++ {
		writeFloat32(t, buf, float32(i+6))
	}
	for i := 0; i < 5; i++ {
		must(uint32(106 + i))
	}

	must(uint16(7))
	writeFloat32(t, buf, 20.1)
	must(uint32(21))
	writeFloat32(t, buf, 20.2)
	writeFloat32(t, buf, 20.3)
	must(uint32(22))
	must(uint32(23))
	must(uint32(24))
	must(uint32(25))
	writeFloat32(t, buf, 20.4)
	buf.Write(make([]byte, 12))
	writeFloat32(t, buf, 26.1)
	writeFloat32(t, buf, 27.2)
	buf.Write(make([]byte, 12))
	writeFloat32(t, buf, 28.3)
	writeFloat32(t, buf, 29.4)
	buf.Write(make([]byte, 25))
	writeFloat32(t, buf, 30.5)
	must(uint32(20260331))
	must(uint32(31))
	writeFloat32(t, buf, 32.6)
	writeFloat32(t, buf, 33.7)
	buf.Write(make([]byte, 24))
	must(uint16(34))
	must(uint8(35))

	wantLen := 291 + codeLen
	if buf.Len() != wantLen {
		t.Fatalf("unexpected ex quote record len: got %d want %d", buf.Len(), wantLen)
	}
	return buf.Bytes()
}

func TestExGetCountParseResponse(t *testing.T) {
	msg := NewExGetCount()
	payload := make([]byte, 31)
	binary.LittleEndian.PutUint32(payload[19:23], 4321)

	if err := msg.ParseResponse(&RespHeader{}, payload); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	if msg.Response().Count != 4321 {
		t.Fatalf("unexpected count: %d", msg.Response().Count)
	}
}

func TestExGetListBuildRequestAndParseResponse(t *testing.T) {
	msg := NewExGetList(&ExGetListRequest{Start: 10, Count: 2})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXLIST {
		t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req ExGetListRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Start != 10 || req.Count != 2 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(10)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(13); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(74); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(9); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(88)); err != nil {
		t.Fatal(err)
	}
	code := make([]byte, 9)
	copy(code, "TSLA")
	buf.Write(code)
	name := make([]byte, 26)
	copy(name, "Tesla")
	buf.Write(name)
	writeFloat32(t, buf, 0.01)
	writeFloat32(t, buf, 100.0)
	for i := 0; i < 8; i++ {
		if err := binary.Write(buf, binary.LittleEndian, uint16(200+i)); err != nil {
			t.Fatal(err)
		}
	}

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}

	reply := msg.Response()
	if reply.Start != 10 || reply.Count != 1 {
		t.Fatalf("unexpected reply header: %+v", reply)
	}
	if len(reply.List) != 1 {
		t.Fatalf("unexpected list len: %d", len(reply.List))
	}
	if reply.List[0].Code != "TSLA" || reply.List[0].Name != "Tesla" {
		t.Fatalf("unexpected list item: %+v", reply.List[0])
	}
}

func TestExGetQuotesListBuildRequestAndParseResponse(t *testing.T) {
	msg := NewExGetQuotesList(&ExGetQuotesListRequest{
		Category:    74,
		SortType:    0x0a,
		Start:       7,
		Count:       3,
		SortReverse: 2,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXQUOTESLIST {
		t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req ExGetQuotesListRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Category != 74 || req.Count != 3 || req.SortReverse != 2 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	buf.Write(buildExQuoteRecord(t, 23, 74, "TSLA"))

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}

	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.Code != "TSLA" || item.Active != 123 || item.Date != "2026-03-31" {
		t.Fatalf("unexpected ex quote item: %+v", item)
	}
	if len(item.BidLevels) != 5 || item.BidLevels[0].Vol != 101 || math.Abs(item.Close-10.3) > 0.001 {
		t.Fatalf("unexpected ex quote levels: %+v", item)
	}
}

func TestExGetQuoteParseResponse(t *testing.T) {
	msg := NewExGetQuote(nil)
	if err := msg.ParseResponse(&RespHeader{}, buildExQuoteRecord(t, 9, 31, "09988")); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	item := msg.Response().Item
	if item.Code != "09988" || item.Category != 31 {
		t.Fatalf("unexpected quote item: %+v", item)
	}
}

func TestExGetQuotes2BuildRequestAndParseResponse(t *testing.T) {
	msg := NewExGetQuotes2(&ExGetQuotesRequest{
		Stocks: []ExStock{{Category: 74, Code: "TSLA"}},
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXQUOTES2 {
		t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}
	if binary.LittleEndian.Uint16(raw[12:14]) != 2 || binary.LittleEndian.Uint16(raw[14:16]) != 3148 {
		t.Fatalf("unexpected quotes2 header body: %x", raw[12:22])
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	buf.Write(buildExQuoteRecord(t, 23, 74, "TSLA"))

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	if len(msg.Response().List) != 1 || msg.Response().List[0].Code != "TSLA" {
		t.Fatalf("unexpected reply: %+v", msg.Response())
	}
}

func TestExGetKLineBuildRequestAndParseResponse(t *testing.T) {
	msg := NewExGetKLine(&ExGetKLineRequest{
		Category: 74,
		Code:     [9]byte{'T', 'S', 'L', 'A'},
		Period:   KLINE_TYPE_DAILY,
		Times:    1,
		Start:    5,
		Count:    2,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXKLINE {
		t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	buf := new(bytes.Buffer)
	buf.WriteByte(74)
	name := make([]byte, 9)
	copy(name, "TSLA")
	buf.Write(name)
	if err := binary.Write(buf, binary.LittleEndian, uint16(KLINE_TYPE_DAILY)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(5)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(20260331)); err != nil {
		t.Fatal(err)
	}
	writeFloat32(t, buf, 100.1)
	writeFloat32(t, buf, 101.2)
	writeFloat32(t, buf, 99.8)
	writeFloat32(t, buf, 100.8)
	writeFloat32(t, buf, 1000000)
	if err := binary.Write(buf, binary.LittleEndian, uint32(8888)); err != nil {
		t.Fatal(err)
	}
	writeFloat32(t, buf, 0)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.List[0].DateTime[:10] != "2026-03-31" || reply.List[0].Vol != 8888 {
		t.Fatalf("unexpected kline item: %+v", reply.List[0])
	}
}

func TestExGetTickChartAndSamplingParseResponse(t *testing.T) {
	tickMsg := NewExGetTickChart(nil)
	tickBuf := new(bytes.Buffer)
	tickBuf.WriteByte(74)
	code := make([]byte, 31)
	copy(code, "TSLA")
	tickBuf.Write(code)
	if err := binary.Write(tickBuf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(tickBuf, binary.LittleEndian, uint16(9*60+31)); err != nil {
		t.Fatal(err)
	}
	writeFloat32(t, tickBuf, 100.5)
	writeFloat32(t, tickBuf, 100.4)
	if err := binary.Write(tickBuf, binary.LittleEndian, uint32(1234)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(tickBuf, binary.LittleEndian, uint32(0)); err != nil {
		t.Fatal(err)
	}

	if err := tickMsg.ParseResponse(&RespHeader{}, tickBuf.Bytes()); err != nil {
		t.Fatalf("tick parse response failed: %v", err)
	}
	if len(tickMsg.Response().List) != 1 || tickMsg.Response().List[0].Time != "09:31" {
		t.Fatalf("unexpected tick reply: %+v", tickMsg.Response())
	}

	samplingMsg := NewExGetChartSampling(nil)
	samplingBuf := new(bytes.Buffer)
	if err := binary.Write(samplingBuf, binary.LittleEndian, uint16(74)); err != nil {
		t.Fatal(err)
	}
	code22 := make([]byte, 22)
	copy(code22, "TSLA")
	samplingBuf.Write(code22)
	for i := 0; i < 8; i++ {
		if err := binary.Write(samplingBuf, binary.LittleEndian, uint16(i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := binary.Write(samplingBuf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	writeFloat32(t, samplingBuf, 100.1)
	writeFloat32(t, samplingBuf, 100.2)

	if err := samplingMsg.ParseResponse(&RespHeader{}, samplingBuf.Bytes()); err != nil {
		t.Fatalf("sampling parse response failed: %v", err)
	}
	if len(samplingMsg.Response().Prices) != 2 || math.Abs(samplingMsg.Response().Prices[1]-100.2) > 0.001 {
		t.Fatalf("unexpected sampling reply: %+v", samplingMsg.Response())
	}
}

func TestExGetBoardListBuildRequestAndParseResponse(t *testing.T) {
	msg := NewExGetBoardList(&ExGetBoardListRequest{
		PageSize:  2,
		BoardType: 4,
		Start:     5,
	})

	raw := mustBuildRequest(t, msg)
	header := readExReqHeader(t, raw)
	if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXBOARDLIST {
		t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
	}

	var req ExGetBoardListRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.PageSize != 2 || req.BoardType != 4 || req.Start != 5 || req.SortOrder != 1 || req.One != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := make([]byte, 22)
	copy(code, "BK0001")
	buf.Write(code)
	name := make([]byte, 44)
	copy(name, "BoardName")
	buf.Write(name)
	writeFloat32(t, buf, 10.1)
	writeFloat32(t, buf, 1.2)
	writeFloat32(t, buf, 9.9)
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	symbolCode := make([]byte, 22)
	copy(symbolCode, "000001")
	buf.Write(symbolCode)
	symbolName := make([]byte, 44)
	copy(symbolName, "PingAn")
	buf.Write(symbolName)
	writeFloat32(t, buf, 12.3)
	writeFloat32(t, buf, 0.4)
	writeFloat32(t, buf, 11.8)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.List[0].Code != "BK0001" || reply.List[0].SymbolCode != "000001" {
		t.Fatalf("unexpected board list item: %+v", reply.List[0])
	}
}

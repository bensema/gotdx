package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestServerMessagesBuildRequestAndParseResponse(t *testing.T) {
	t.Run("heartbeat", func(t *testing.T) {
		msg := NewHeartBeat()
		raw := mustBuildRequest(t, msg)
		header := readReqHeader(t, raw)
		if header.Method != KMSG_HEARTBEAT {
			t.Fatalf("unexpected method: %#x", header.Method)
		}
		payload := make([]byte, 10)
		binary.LittleEndian.PutUint32(payload[6:10], 20250512)
		if err := msg.ParseResponse(&RespHeader{}, payload); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		if msg.Response().Date != 20250512 {
			t.Fatalf("unexpected heartbeat date: %d", msg.Response().Date)
		}
	})

	t.Run("exchange_announcement", func(t *testing.T) {
		msg := NewExchangeAnnouncement()
		if err := msg.ParseResponse(&RespHeader{}, append([]byte{1}, []byte("hello")...)); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		if msg.Response().Version != 1 || msg.Response().Content != "hello" {
			t.Fatalf("unexpected reply: %+v", msg.Response())
		}
	})

	t.Run("announcement", func(t *testing.T) {
		msg := NewAnnouncement()
		buf := new(bytes.Buffer)
		buf.WriteByte(1)
		if err := binary.Write(buf, binary.LittleEndian, uint32(20260411)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(5)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(6)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(7)); err != nil {
			t.Fatal(err)
		}
		buf.WriteString("title")
		buf.WriteString("author")
		buf.WriteString("content")
		if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if !reply.HasContent || reply.ExpireDate != "2026-04-11" || reply.Title != "title" || reply.Author != "author" || reply.Content != "content" {
			t.Fatalf("unexpected reply: %+v", reply)
		}
	})

	t.Run("info", func(t *testing.T) {
		msg := NewInfo()
		payload := make([]byte, 427)
		binary.LittleEndian.PutUint32(payload[0:4], 123)
		copy(payload[16:71], []byte("server-info"))
		copy(payload[81:336], []byte("server-content"))
		copy(payload[336:356], []byte("sign"))
		binary.LittleEndian.PutUint16(payload[389:391], 88)
		binary.LittleEndian.PutUint16(payload[395:397], 1)
		binary.LittleEndian.PutUint32(payload[397:401], 20260411)
		binary.LittleEndian.PutUint32(payload[401:405], 93015)
		if err := msg.ParseResponse(&RespHeader{}, payload); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Delay != 123 || reply.Info != "server-info" || reply.Content != "server-content" || reply.ServerSign != "sign" {
			t.Fatalf("unexpected info reply: %+v", reply)
		}
		if reply.TimeNow != "2026-04-11 09:30:15" || reply.Region != 88 || reply.MaybeSwitch != 1 {
			t.Fatalf("unexpected info metadata: %+v", reply)
		}
	})
}

func TestGetSecurityListOldBuildRequestAndParseResponse(t *testing.T) {
	msg := NewGetSecurityListOld(&GetSecurityListOldRequest{Market: 1, Start: 8})

	raw := mustBuildRequest(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_SECURITYLIST_OLD {
		t.Fatalf("unexpected method: %#x", header.Method)
	}
	var req GetSecurityListOldRequest
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Market != 1 || req.Start != 8 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	code := [6]byte{'6', '0', '0', '0', '0', '0'}
	name := [8]byte{'T', 'E', 'S', 'T'}
	if err := binary.Write(buf, binary.LittleEndian, code); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(100)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(0)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(2); err != nil {
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

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	item := msg.Response().List[0]
	if item.Code != "600000" || item.Name != "TEST" || item.DecimalPoint != 2 || math.Abs(item.PreClose-12.34) > 0.001 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestGetHistoryTransactionDataWithTransBuildRequestAndParseResponse(t *testing.T) {
	msg := NewGetHistoryTransactionDataWithTrans(&GetHistoryTransactionDataRequest{
		Date:   20260411,
		Market: 1,
		Code:   [6]byte{'6', '0', '0', '0', '0', '0'},
		Start:  0,
		Count:  1,
	})
	raw := mustBuildRequest(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_TRANSACTIONDATA_TRANS {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, math.Float32bits(12.3)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint16(570)); err != nil {
		t.Fatal(err)
	}
	buf.Write(encodePrice(1234))
	buf.Write(encodePrice(100))
	buf.Write(encodePrice(2))
	if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
		t.Fatal(err)
	}
	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || math.Abs(reply.PreClose-12.3) > 0.001 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.Time != "09:30" || item.Action != "SELL" || math.Abs(item.Price-12.34) > 0.001 {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestGetQuotesEncryptBuildRequestAndParseResponse(t *testing.T) {
	msg := NewGetQuotesEncrypt(&GetQuotesEncryptRequest{Stocks: []Stock{{Market: 1, Code: "600000"}}})
	raw := mustBuildRequest(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_QUOTESENCRYPT {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var count uint16 = 1
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, count); err != nil {
		t.Fatal(err)
	}
	if err := payload.WriteByte(1); err != nil {
		t.Fatal(err)
	}
	code := [6]byte{'6', '0', '0', '0', '0', '0'}
	if err := binary.Write(payload, binary.LittleEndian, code); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(payload, binary.LittleEndian, uint16(9)); err != nil {
		t.Fatal(err)
	}
	payload.Write(encodePrice(1234))
	payload.Write(encodePrice(10))
	payload.Write(encodePrice(5))
	payload.Write(encodePrice(20))
	payload.Write(encodePrice(-10))
	if err := binary.Write(payload, binary.LittleEndian, uint32(93015)); err != nil {
		t.Fatal(err)
	}
	payload.Write(encodePrice(0))
	payload.Write(encodePrice(1000))
	payload.Write(encodePrice(50))
	if err := binary.Write(payload, binary.LittleEndian, math.Float32bits(12345.6)); err != nil {
		t.Fatal(err)
	}
	payload.Write(encodePrice(100))
	payload.Write(encodePrice(90))
	payload.Write(encodePrice(80))
	payload.Write(encodePrice(70))
	for i := 0; i < 5; i++ {
		payload.Write(encodePrice(i))
		payload.Write(encodePrice(i + 1))
		payload.Write(encodePrice(100 + i))
		payload.Write(encodePrice(200 + i))
	}
	payload.Write(make([]byte, 10))
	for i := 0; i < 24; i++ {
		payload.Write(encodePrice(i))
	}
	xor := payload.Bytes()
	for i := range xor {
		xor[i] ^= 0x93
	}
	if err := msg.ParseResponse(&RespHeader{}, xor); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 1 || len(reply.List) != 1 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	item := reply.List[0]
	if item.Code != "600000" || item.Time != "09:30:15" || math.Abs(item.Close-12.34) > 0.001 || len(item.BidLevels) != 5 {
		t.Fatalf("unexpected encrypted quote item: %+v", item)
	}
}

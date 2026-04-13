package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

func TestRawMainProtocolBuildRequestAndParseResponse(t *testing.T) {
	t.Run("todo_b", func(t *testing.T) {
		msg := NewTodoB()
		raw := mustBuildRequest(t, msg)
		header := readReqHeader(t, raw)
		if header.Method != KMSG_TODOB {
			t.Fatalf("unexpected method: %#x", header.Method)
		}
		if len(raw) <= 12 {
			t.Fatalf("unexpected raw length: %d", len(raw))
		}
		if err := msg.ParseResponse(&RespHeader{}, []byte{0x01, 0x02, 0x03}); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Length != 3 || reply.Hex != "010203" {
			t.Fatalf("unexpected raw reply: %+v", reply)
		}
	})

	t.Run("client_26ad", func(t *testing.T) {
		msg := NewClient26AD()
		raw := mustBuildRequest(t, msg)
		header := readReqHeader(t, raw)
		if header.Method != KMSG_CLIENT26AD {
			t.Fatalf("unexpected method: %#x", header.Method)
		}
		if header.PkgLen1 != header.PkgLen2 || header.PkgLen1 <= 2 {
			t.Fatalf("unexpected pkg lens: %+v", header)
		}
	})
}

func TestGetSecurityFeature452BuildRequestAndParseResponse(t *testing.T) {
	msg := NewGetSecurityFeature452(&GetSecurityFeature452Request{Start: 9, Count: 2})

	raw := mustBuildRequest(t, msg)
	header := readReqHeader(t, raw)
	if header.Method != KMSG_SECURITYFEATURE452 {
		t.Fatalf("unexpected method: %#x", header.Method)
	}

	var req GetSecurityFeature452Request
	if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
		t.Fatalf("read request failed: %v", err)
	}
	if req.Start != 9 || req.Count != 2 || req.One != 1 {
		t.Fatalf("unexpected request: %+v", req)
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint16(2)); err != nil {
		t.Fatal(err)
	}
	if err := buf.WriteByte(1); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(600000)); err != nil {
		t.Fatal(err)
	}
	writeFloat32(t, buf, 1.25)
	writeFloat32(t, buf, 2.5)
	if err := buf.WriteByte(0); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(buf, binary.LittleEndian, uint32(1)); err != nil {
		t.Fatal(err)
	}
	writeFloat32(t, buf, 3.75)
	writeFloat32(t, buf, 4.5)

	if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
		t.Fatalf("parse response failed: %v", err)
	}
	reply := msg.Response()
	if reply.Count != 2 || len(reply.List) != 2 {
		t.Fatalf("unexpected reply: %+v", reply)
	}
	if reply.List[0].Code != "600000" || math.Abs(reply.List[1].P2-4.5) > 0.001 {
		t.Fatalf("unexpected list: %+v", reply.List)
	}
}

func TestExExperimentalMessagesBuildRequestAndParseResponse(t *testing.T) {
	t.Run("23f6", func(t *testing.T) {
		msg := NewExGetListExtra(&ExGetListExtraRequest{A: 1, B: 2, Count: 3})

		raw := mustBuildRequest(t, msg)
		header := readExReqHeader(t, raw)
		if header.Head != 0x01 || binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXLIST_EXTRA {
			t.Fatalf("unexpected ex request header: head=%#x method=%#x", header.Head, binary.LittleEndian.Uint16(raw[10:12]))
		}
		var req ExGetListExtraRequest
		if err := binary.Read(bytes.NewReader(raw[12:]), binary.LittleEndian, &req); err != nil {
			t.Fatalf("read request failed: %v", err)
		}
		if req.A != 1 || req.B != 2 || req.Count != 3 {
			t.Fatalf("unexpected request: %+v", req)
		}

		buf := new(bytes.Buffer)
		if err := binary.Write(buf, binary.LittleEndian, uint32(7)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
			t.Fatal(err)
		}
		if err := buf.WriteByte(74); err != nil {
			t.Fatal(err)
		}
		code := [8]byte{'T', 'S', 'L', 'A'}
		if err := binary.Write(buf, binary.LittleEndian, code); err != nil {
			t.Fatal(err)
		}
		if err := buf.WriteByte(9); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 12; i++ {
			if err := binary.Write(buf, binary.LittleEndian, uint16(100+i)); err != nil {
				t.Fatal(err)
			}
		}

		if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Start != 7 || reply.Count != 1 || len(reply.List) != 1 {
			t.Fatalf("unexpected reply: %+v", reply)
		}
		if reply.List[0].Code != "TSLA" || len(reply.List[0].Values) != 12 {
			t.Fatalf("unexpected item: %+v", reply.List[0])
		}
	})

	t.Run("2487", func(t *testing.T) {
		msg := NewExExperiment2487(&ExExperiment2487Request{Category: 74, Code: [23]byte{'T', 'S', 'L', 'A'}})

		raw := mustBuildRequest(t, msg)
		if binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXQUOTES_EXPERIMENT1 {
			t.Fatalf("unexpected method: %#x", binary.LittleEndian.Uint16(raw[10:12]))
		}

		buf := new(bytes.Buffer)
		if err := buf.WriteByte(74); err != nil {
			t.Fatal(err)
		}
		code := [23]byte{'T', 'S', 'L', 'A'}
		if err := binary.Write(buf, binary.LittleEndian, code); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(99)); err != nil {
			t.Fatal(err)
		}
		writeFloat32(t, buf, 10.1)
		writeFloat32(t, buf, 10.2)
		writeFloat32(t, buf, 10.3)
		writeFloat32(t, buf, 10.0)
		writeFloat32(t, buf, 10.4)
		writeFloat32(t, buf, 10.5)
		writeFloat32(t, buf, 10.6)
		if err := binary.Write(buf, binary.LittleEndian, uint32(101)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(12)); err != nil {
			t.Fatal(err)
		}
		writeFloat32(t, buf, 1234.5)
		buf.Write([]byte{0xaa, 0xbb})

		if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Code != "TSLA" || reply.Active != 99 || reply.TailHex != "aabb" {
			t.Fatalf("unexpected reply: %+v", reply)
		}
	})

	t.Run("2488", func(t *testing.T) {
		msg := NewExExperiment2488(&ExExperiment2488Request{Category: 31, Code: [23]byte{'0', '9', '9', '8', '8'}, Mode: 55})

		raw := mustBuildRequest(t, msg)
		if binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXQUOTES_EXPERIMENT2 {
			t.Fatalf("unexpected method: %#x", binary.LittleEndian.Uint16(raw[10:12]))
		}

		buf := new(bytes.Buffer)
		if err := buf.WriteByte(31); err != nil {
			t.Fatal(err)
		}
		code := make([]byte, 35)
		copy(code, "09988")
		if _, err := buf.Write(code); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(7)); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 6; i++ {
			if err := binary.Write(buf, binary.LittleEndian, uint16(10+i)); err != nil {
				t.Fatal(err)
			}
		}

		if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Code != "09988" || reply.Count != 1 || len(reply.List) != 1 {
			t.Fatalf("unexpected reply: %+v", reply)
		}
		if reply.List[0].ID != 7 || len(reply.List[0].Values) != 6 {
			t.Fatalf("unexpected item: %+v", reply.List[0])
		}
	})

	t.Run("2489", func(t *testing.T) {
		msg := NewExGetKLine2(&ExGetKLine2Request{
			Category: 74,
			Code:     [23]byte{'T', 'S', 'L', 'A'},
			Period:   4,
			Times:    1,
			Start:    0,
			Count:    2,
		})

		raw := mustBuildRequest(t, msg)
		if binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXKLINE2 {
			t.Fatalf("unexpected method: %#x", binary.LittleEndian.Uint16(raw[10:12]))
		}

		buf := new(bytes.Buffer)
		if err := buf.WriteByte(74); err != nil {
			t.Fatal(err)
		}
		name := [23]byte{'T', 'e', 's', 'l', 'a'}
		if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(4)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(20260411)); err != nil {
			t.Fatal(err)
		}
		writeFloat32(t, buf, 10)
		writeFloat32(t, buf, 11)
		writeFloat32(t, buf, 9)
		writeFloat32(t, buf, 10.5)
		writeFloat32(t, buf, 1200.5)
		if err := binary.Write(buf, binary.LittleEndian, uint32(123)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(0)); err != nil {
			t.Fatal(err)
		}

		if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Name != "Tesla" || reply.Count != 1 || len(reply.List) != 1 {
			t.Fatalf("unexpected reply: %+v", reply)
		}
		if reply.List[0].DateTime != "2026-04-11 15:00:00" || math.Abs(reply.List[0].Close-10.5) > 0.001 {
			t.Fatalf("unexpected item: %+v", reply.List[0])
		}
	})

	t.Run("2562", func(t *testing.T) {
		msg := NewExMapping2562(&ExMapping2562Request{Market: 47, Start: 5, Count: 2})

		raw := mustBuildRequest(t, msg)
		if binary.LittleEndian.Uint16(raw[10:12]) != KMSG_EXMAPPING2562 {
			t.Fatalf("unexpected method: %#x", binary.LittleEndian.Uint16(raw[10:12]))
		}

		buf := new(bytes.Buffer)
		if err := binary.Write(buf, binary.LittleEndian, uint16(1)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(74)); err != nil {
			t.Fatal(err)
		}
		name := [23]byte{'U', 'S', ' ', 'S', 't', 'o', 'c', 'k'}
		if err := binary.Write(buf, binary.LittleEndian, name); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(8)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint32(99)); err != nil {
			t.Fatal(err)
		}
		if err := buf.WriteByte(1); err != nil {
			t.Fatal(err)
		}
		writeFloat32(t, buf, 1.1)
		writeFloat32(t, buf, 2.2)
		writeFloat32(t, buf, 3.3)
		if err := binary.Write(buf, binary.LittleEndian, uint16(4)); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(buf, binary.LittleEndian, uint16(5)); err != nil {
			t.Fatal(err)
		}

		if err := msg.ParseResponse(&RespHeader{}, buf.Bytes()); err != nil {
			t.Fatalf("parse response failed: %v", err)
		}
		reply := msg.Response()
		if reply.Count != 1 || len(reply.List) != 1 {
			t.Fatalf("unexpected reply: %+v", reply)
		}
		if reply.List[0].Name != "US Stock" || reply.List[0].Index != 99 || math.Abs(reply.List[0].Code3-3.3) > 0.001 {
			t.Fatalf("unexpected item: %+v", reply.List[0])
		}
	})
}

package gotdx

import "testing"

func TestMACFieldBitmapPresets(t *testing.T) {
	// 默认板块成员行情位图的字节序列与协议约定一致。
	defaultBitmap := DefaultMACBoardMembersQuotesFieldBitmap()
	expectedDefault := [20]byte{0xff, 0xfc, 0xe1, 0xcc, 0x3f, 0x08, 0x03, 0x01}
	if defaultBitmap != expectedDefault {
		t.Fatalf("default bitmap mismatch: got %x want %x", defaultBitmap, expectedDefault)
	}

	// `MACPresetCommon` 预设展开后的固定字节序列。
	common := MACFieldBitmap(MACPresetCommon)
	expectedCommon := [20]byte{0xff, 0xfc, 0xa1, 0xcc, 0x3f, 0x28, 0x03, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03}
	if common != expectedCommon {
		t.Fatalf("common preset literal mismatch: got %x want %x", common, expectedCommon)
	}

	// 预设编号与单个字段位可叠加组合成自定义位图。
	custom := MACFieldBitmap(MACPresetOHLC, MACFieldAHCode)
	expected := MACBoardMembersQuotesFieldBitmapFromBits(1, 2, 3, 4, 0x4a)
	if custom != expected {
		t.Fatalf("custom bitmap mismatch: got %x want %x", custom, expected)
	}

	// `MACPresetDebug` 选中全部字段位，每个字节均为 `0xff`。
	debug := MACFieldBitmap(MACPresetDebug)
	for i, b := range debug {
		if b != 0xff {
			t.Fatalf("debug byte %d = %#x", i, b)
		}
	}
}

func TestMACFieldAliases(t *testing.T) {
	// 历史别名常量必须与现行字段位保持同一取值，避免旧调用方错位。
	if MACFieldBid2Volume != MACFieldLimitUpCount {
		t.Fatal("bid2 volume alias should match limit up count")
	}
	if MACFieldAsk5Volume != MACFieldDownCount {
		t.Fatal("ask5 volume alias should match down count")
	}
	if MACFieldPreIPOV != MACFieldPreIOPV {
		t.Fatal("legacy pre IPOV alias should match pre IOPV")
	}
	if MACFieldConstantNegOne != MACFieldSafetyScore {
		t.Fatal("legacy safety-score alias should match latest field")
	}
}

func TestMACBoardMembersQuotesRequestBitmap(t *testing.T) {
	// 请求位图在字段位图之外追加板块过滤标志位与扩展控制位。
	bitmap := MACFieldBitmap(MACFieldMainNetRatio, MACFieldChangeAt1000)
	requestBitmap := MACBoardMembersQuotesRequestBitmap(bitmap, MACFilterKCB, MACFilterST, MACFilterBJ)

	// 字段位 `0x6c` 落在第 `0x6c/8` 字节，位内偏移为 `0x6c%8`。
	if requestBitmap[0x6c/8]&(1<<(0x6c%8)) == 0 {
		t.Fatalf("main net ratio bit missing: %x", requestBitmap)
	}
	// 字段位 `0x90` 落在第 `0x90/8` 字节，位内偏移为 `0x90%8`。
	if requestBitmap[0x90/8]&(1<<(0x90%8)) == 0 {
		t.Fatalf("change-at-1000 bit missing: %x", requestBitmap)
	}
	if requestBitmap[17] != MACFilterFlags(MACFilterKCB, MACFilterST, MACFilterBJ) { // 第 17 字节为板块过滤标志位
		t.Fatalf("unexpected filter byte: %#x", requestBitmap[17])
	}
	if requestBitmap[19]&1 == 0 { // 第 19 字节最低位为扩展控制位
		t.Fatalf("extended control bit missing: %#x", requestBitmap[19])
	}
}

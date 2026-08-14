package proto

import "testing"

// TestUtf8ToGbkTruncatedName 覆盖固定宽度名称字段按字节截断的场景：
// 服务器在 16 字节 GBK 字段内截断名称，末字节是半个汉字，Utf8ToGbk 不应透出 U+FFFD。
func TestUtf8ToGbkTruncatedName(t *testing.T) {
	t.Run("main market 16-byte field", func(t *testing.T) {
		// "红利低波50ETF南方" GBK 为 17 字节，截断到 16 字节后末字节 b7 是"方"(b7bd) 的前导字节。
		raw := []byte{0xba, 0xec, 0xc0, 0xfb, 0xb5, 0xcd, 0xb2, 0xa8, 0x35, 0x30, 0x45, 0x54, 0x46, 0xc4, 0xcf, 0xb7}
		if got := Utf8ToGbk(raw); got != "红利低波50ETF南" {
			t.Fatalf("Utf8ToGbk() = %q, want %q", got, "红利低波50ETF南")
		}
	})

	t.Run("extended market 26-byte field padded with zero", func(t *testing.T) {
		raw := append([]byte{0xba, 0xec, 0xc0, 0xfb, 0xb5, 0xcd, 0xb2, 0xa8, 0x35, 0x30, 0x45, 0x54, 0x46, 0xc4, 0xcf, 0xb7}, make([]byte, 10)...)
		if got := Utf8ToGbk(raw); got != "红利低波50ETF南" {
			t.Fatalf("Utf8ToGbk() = %q, want %q", got, "红利低波50ETF南")
		}
	})

	t.Run("complete name preserved", func(t *testing.T) {
		raw := []byte{0xba, 0xec, 0xc0, 0xfb, 0xb5, 0xcd, 0xb2, 0xa8, 0x35, 0x30, 0x45, 0x54, 0x46, 0xc4, 0xcf, 0xb7, 0xbd}
		if got := Utf8ToGbk(raw); got != "红利低波50ETF南方" {
			t.Fatalf("Utf8ToGbk() = %q, want %q", got, "红利低波50ETF南方")
		}
	})

	t.Run("plain text untouched", func(t *testing.T) {
		if got := Utf8ToGbk([]byte("example")); got != "example" {
			t.Fatalf("Utf8ToGbk() = %q, want %q", got, "example")
		}
	})
}

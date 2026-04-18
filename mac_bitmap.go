package gotdx

// DefaultMACBoardMembersQuotesFieldBitmap 返回与当前稳定成分报价接口一致的默认字段位图。
func DefaultMACBoardMembersQuotesFieldBitmap() [20]byte {
	return [20]byte{
		0xff, 0xfc, 0xe1, 0xcc, 0x3f, 0x08, 0x03, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
}

// FullMACBoardMembersQuotesFieldBitmap 返回 20 字节全 1 位图，适合实验性全字段请求。
func FullMACBoardMembersQuotesFieldBitmap() [20]byte {
	var bitmap [20]byte
	for i := range bitmap {
		bitmap[i] = 0xff
	}
	return bitmap
}

// MACBoardMembersQuotesFieldBitmapFromBits 根据 bit 列表组装 20 字节字段位图。
func MACBoardMembersQuotesFieldBitmapFromBits(bits ...int) [20]byte {
	var bitmap [20]byte
	for _, bit := range bits {
		if bit < 0 || bit >= len(bitmap)*8 {
			continue
		}
		bitmap[bit/8] |= 1 << uint(bit%8)
	}
	return bitmap
}

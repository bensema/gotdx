package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/bensema/gotdx/types"
	"github.com/spf13/cast"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Utf8ToGbk 把服务端返回的 GBK 兼容 GB18030 字节流解码为 UTF-8 字符串，并清理填充与非法字符。
//
// 参数 `text` 为服务端原始 GBK 字节流；返回清理后的 UTF-8 字符串。
// 函数名沿用历史命名，实际行为是 GBK 解码为 UTF-8；解码后删除空字节填充，剔除字段尾部半个
// 汉字解码出的替换符，并去除首尾控制字符。
func Utf8ToGbk(text []byte) string {

	r := bytes.NewReader(text)

	decoder := transform.NewReader(r, simplifiedchinese.GBK.NewDecoder()) // GB18030

	content, _ := io.ReadAll(decoder)

	result := strings.ReplaceAll(string(content), string([]byte{0x00}), "")
	// 定长字段按字节截断时末字节可能是半个 GBK 字符，替换符只会出现在字段尾部。
	result = strings.TrimRight(result, "\uFFFD")
	return strings.TrimFunc(result, func(r rune) bool {
		return unicode.IsControl(r)
	})
}

// GetTime 按 K 线周期类别解码四字节行情时间戳，返回本地时间。
//
// 参数 `data` 为四字节时间戳，`category` 为 K 线周期类别；返回解码后的本地时间。
// 分钟级周期前两字节为位压缩日期、后两字节为当日累计分钟；日及以上周期四字节为年月日打包值，
// 没有时分信息，统一取收盘时点。
func GetTime(data [4]byte, category uint16) time.Time {
	newCategory := cast.ToInt(category)
	switch newCategory {
	case types.KLINE_TYPE_EXHQ_1MIN, types.KLINE_TYPE_1MIN, types.KLINE_TYPE_5MIN, types.KLINE_TYPE_15MIN, types.KLINE_TYPE_30MIN, types.KLINE_TYPE_1HOUR:

		// 分钟周期：前两字节为位压缩日期，后两字节为当日累计分钟，均解码。
		yearMonthDay := Uint16(data[:2])
		hourMinute := Uint16(data[2:4])
		year := int(yearMonthDay>>11 + 2004)
		month := yearMonthDay % 2048 / 100
		day := int((yearMonthDay % 2048) % 100)
		hour := int(hourMinute / 60)
		minute := int(hourMinute % 60)
		return time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)

	default:
		// fmt.Println("类型错误", Uint32(data[:4]))
		// 日及以上周期：四字节为年月日打包整数，无时分信息，统一取收盘时点十五点整。
		yearMonthDay := Uint32(data[:4])
		year := int(yearMonthDay / 10000)
		month := int((yearMonthDay % 10000) / 100)
		day := int(yearMonthDay % 100)
		return time.Date(year, time.Month(month), day, 15, 0, 0, 0, time.Local)
	}
}

// Uint16 把字节切片解码为十六位无符号整数；切片长度不足时按高位缺失处理。
//
// 参数 i 为字节切片；返回解码后的十六位无符号整数。
func Uint16(i []byte) uint16 {
	i = Reverse(i)
	return uint16(Uint64(i))
}

// Uint32 把字节切片解码为三十二位无符号整数；切片长度不足时按高位缺失处理。
//
// 参数 i 为字节切片；返回解码后的三十二位无符号整数。
func Uint32(i []byte) uint32 {
	i = Reverse(i)
	return uint32(Uint64(i))
}

// Reverse 返回字节切片顺序反转后的新切片，不修改原切片。
//
// 参数 `data` 为待反转字节切片；返回反转后的新切片。
func Reverse(data []byte) []byte {
	x := make([]byte, len(data))
	for i, v := range data {
		x[len(data)-i-1] = v
	}
	return x
}

// getData 解码通达信变长有符号整数；首字节低六位为数据位，后续字节低七位按七位步长拼接，
// 最高位为续接标志，首字节符号位置位时结果取负。
//
// 参数 `data` 为变长编码字节段；返回解码后的三十二位有符号整数。
func getData(data []byte) (res int32) {

	for i := range data {
		switch i {
		case 0:
			res += int32(data[0] & 0x3F)

		default:
			res += int32(data[i]&0x7F) << uint8(6+(i-1)*7)
		}

		if data[i]&0x80 == 0 {
			break
		}
	}

	if len(data) > 0 && data[0]&0x40 > 0 {
		res = -res
	}

	return
}

// GetPrice 从变长编码字节流中解析一个整数价格，返回编码结束后剩余的字节切片与解码结果。
//
// 参数 bs 为变长编码字节流；返回剩余切片与解码价格，找不到结束标志时返回原切片与零值。
func GetPrice(bs []byte) ([]byte, int) {
	for i := range bs {
		if bs[i]&0x80 == 0 {
			return bs[i+1:], cast.ToInt(getData(bs[:i+1]))
		}
	}
	return bs, 0
}

// getVolume 解析服务端压缩传输的成交量，算法与 `proto.go` 中的 `getvolume` 同源；
// 四字节拆为一个指数字节与三个尾数字节，各段以二的幂为权重分别重建后求和。
//
// 参数 `val` 为服务端四字节压缩编码的成交量整数；返回还原后的浮点成交量。
// 该算法由历史 C 实现平移移植，指数偏移常量沿用原实现。
func getVolume(val uint32) (volume float64) {
	ivol := int32(val)
	logpoint := ivol >> (8 * 3)
	//hheax := ivol >> (8 * 3)          // [3]
	hleax := (ivol >> (8 * 2)) & 0xff // [2]
	lheax := (ivol >> 8) & 0xff       //[1]
	lleax := ivol & 0xff              //[0]

	//dbl_1 := 1.0
	//dbl_2 := 2.0
	//dbl_128 := 128.0

	dwEcx := logpoint*2 - 0x7f
	dwEdx := logpoint*2 - 0x86
	dwEsi := logpoint*2 - 0x8e
	dwEax := logpoint*2 - 0x96
	tmpEax := dwEcx
	if dwEcx < 0 {
		tmpEax = -dwEcx
	} else {
		tmpEax = dwEcx
	}

	dbl_xmm6 := 0.0
	dbl_xmm6 = math.Pow(2.0, float64(tmpEax))
	if dwEcx < 0 {
		dbl_xmm6 = 1.0 / dbl_xmm6
	}

	dbl_xmm4 := 0.0
	dbl_xmm0 := 0.0

	if hleax > 0x80 {
		tmpdbl_xmm3 := 0.0
		//tmpdbl_xmm1 := 0.0
		dwtmpeax := dwEdx + 1
		tmpdbl_xmm3 = math.Pow(2.0, float64(dwtmpeax))
		dbl_xmm0 = math.Pow(2.0, float64(dwEdx)) * 128.0
		dbl_xmm0 += float64(hleax&0x7f) * tmpdbl_xmm3
		dbl_xmm4 = dbl_xmm0
	} else {
		if dwEdx >= 0 {
			dbl_xmm0 = math.Pow(2.0, float64(dwEdx)) * float64(hleax)
		} else {
			dbl_xmm0 = (1 / math.Pow(2.0, float64(dwEdx))) * float64(hleax)
		}
		dbl_xmm4 = dbl_xmm0
	}

	dbl_xmm3 := math.Pow(2.0, float64(dwEsi)) * float64(lheax)
	dbl_xmm1 := math.Pow(2.0, float64(dwEax)) * float64(lleax)
	// 中段尾数最高位置位时，低位两段权重翻倍。
	if (hleax & 0x80) > 0 {
		dbl_xmm3 *= 2.0
		dbl_xmm1 *= 2.0
	}
	volume = dbl_xmm6 + dbl_xmm4 + dbl_xmm3 + dbl_xmm1
	return
}

// padding 在字节切片左侧补零到指定长度；长度已达标时原样返回。
//
// 参数 `bytes` 为待补齐切片，`length` 为目标字节长度；返回补齐后的切片。
func padding(bytes []byte, length int) []byte {
	if len(bytes) < length {
		return append(make([]byte, length-len(bytes)), bytes...)
	}
	return bytes
}

// toUint64 把左侧补齐后的八字节切片读取为六十四位无符号整数；非字节输入返回零。
//
// 参数 i 为待解析值，仅接受字节切片；返回六十四位无符号整数，类型不匹配时返回零。
func toUint64(i any) uint64 {
	if b, ok := i.([]byte); ok {
		return binary.BigEndian.Uint64(padding(b, 8))
	}
	return 0
}

// toInt64 把左侧补齐后的八字节切片读取为六十四位有符号整数；非字节输入返回零。
//
// 参数 i 为待解析值，仅接受字节切片；返回六十四位有符号整数，类型不匹配时返回零。
func toInt64(i any) int64 {
	if b, ok := i.([]byte); ok {
		return int64(binary.BigEndian.Uint64(padding(b, 8)))
	}
	return 0
}

// Uint64 把字节切片补齐至八字节后解析为六十四位无符号整数；非字节输入返回零。
//
// 参数 i 为待解析值，仅接受字节切片；返回六十四位无符号整数。
func Uint64(i any) uint64 {
	return toUint64(i)
}

// Int64 把字节切片补齐至八字节后解析为六十四位有符号整数；非字节输入返回零。
//
// 参数 i 为待解析值，仅接受字节切片；返回六十四位有符号整数。
func Int64(i any) int64 {
	return toInt64(i)
}

// Int 基于 `Int64` 的解析结果转换为系统位宽整数。
//
// 参数 i 为待解析值；返回系统位宽整数，类型不匹配时返回零。
func Int(i any) int {
	return int(Int64(i))
}

package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cast"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func Utf8ToGbk(text []byte) string {

	r := bytes.NewReader(text)

	decoder := transform.NewReader(r, simplifiedchinese.GBK.NewDecoder()) //GB18030

	content, _ := io.ReadAll(decoder)

	result := strings.ReplaceAll(string(content), string([]byte{0x00}), "")
	return strings.TrimFunc(result, func(r rune) bool {
		return unicode.IsControl(r)
	})
}

func GetTime(data [4]byte, category uint16) time.Time {
	newCategory := cast.ToInt(category)
	switch newCategory {
	case KLINE_TYPE_EXHQ_1MIN, KLINE_TYPE_1MIN, KLINE_TYPE_5MIN, KLINE_TYPE_15MIN, KLINE_TYPE_30MIN, KLINE_TYPE_1HOUR:

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
		yearMonthDay := Uint32(data[:4])
		year := int(yearMonthDay / 10000)
		month := int((yearMonthDay % 10000) / 100)
		day := int(yearMonthDay % 100)
		return time.Date(year, time.Month(month), day, 15, 0, 0, 0, time.Local)
	}
}

func Uint16(i []byte) uint16 {
	i = Reverse(i)
	return uint16(Uint64(i))
}

func Uint32(i []byte) uint32 {
	i = Reverse(i)
	return uint32(Uint64(i))
}

func Reverse(data []byte) []byte {
	x := make([]byte, len(data))
	for i, v := range data {
		x[len(data)-i-1] = v
	}
	return x
}

func getData(data []byte) (res int32) {

	for i := range data {
		switch i {
		case 0:
			//取后6位
			res += int32(data[0] & 0x3F)

		default:
			//取后7位
			res += int32(data[i]&0x7F) << uint8(6+(i-1)*7)

		}

		//判断是否有后续数据
		if data[i]&0x80 == 0 {
			break
		}
	}

	//第一字节的第二位为1表示为负数
	if len(data) > 0 && data[0]&0x40 > 0 {
		res = -res
	}

	return
}

func GetPrice(bs []byte) ([]byte, int) {
	for i := range bs {
		if bs[i]&0x80 == 0 {
			return bs[i+1:], cast.ToInt(getData(bs[:i+1]))
		}
	}
	return bs, 0
}

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
	if (hleax & 0x80) > 0 {
		dbl_xmm3 *= 2.0
		dbl_xmm1 *= 2.0
	}
	volume = dbl_xmm6 + dbl_xmm4 + dbl_xmm3 + dbl_xmm1
	return
}

func padding(bytes []byte, length int) []byte {
	if len(bytes) < length {
		return append(make([]byte, length-len(bytes)), bytes...)
	}
	return bytes
}

func toUint64(i any) uint64 {
	if b, ok := i.([]byte); ok {
		return binary.BigEndian.Uint64(padding(b, 8))
	}
	return 0
}

func toInt64(i any) int64 {
	if b, ok := i.([]byte); ok {
		return int64(binary.BigEndian.Uint64(padding(b, 8)))
	}
	return 0
}

func Uint64(i any) uint64 {
	return toUint64(i)
}

func Int64(i any) int64 {
	return toInt64(i)
}

func Int(i any) int {
	return int(Int64(i))
}

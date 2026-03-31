package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync/atomic"
	"time"
)

const (
	MessageHeaderBytes = 0x10
	MessageMaxBytes    = 1 << 15
)

const (
	KMSG_CMD1                   = 0x000d // 建立链接
	KMSG_CMD2                   = 0x0fdb // 建立链接
	KMSG_PING                   = 0x0015 // 测试连接
	KMSG_HEARTBEAT              = 0x0004 // 心跳
	KMSG_SECURITYCOUNT          = 0x044e // 证券数量
	KMSG_BLOCKINFOMETA          = 0x02c5 // 板块文件信息
	KMSG_BLOCKINFO              = 0x06b9 // 板块文件
	KMSG_COMPANYCATEGORY        = 0x02cf // 公司信息文件信息
	KMSG_COMPANYCONTENT         = 0x02d0 // 公司信息描述
	KMSG_FINANCEINFO            = 0x0010 // 财务信息
	KMSG_HISTORYMINUTETIMEDATE  = 0x0feb // 历史分时信息
	KMSG_HISTORYTRANSACTIONDATA = 0x0fb5 // 历史分笔成交信息
	KMSG_INDEXBARS              = 0x0523 // 指数K线
	KMSG_VOLUMEPROFILE          = 0x051a // 成交分布
	KMSG_INDEXMOMENTUM          = 0x051c // 指数动量
	KMSG_INDEXINFO              = 0x051d // 指数概况
	KMSG_SECURITYBARS           = 0x0523 // 股票K线
	KMSG_SECURITYBARS_OFFSET    = 0x052d // 偏移K线
	KMSG_MINUTETIMEDATA         = 0x0537 // 分时数据
	KMSG_SECURITYLIST           = 0x044d // 证券列表
	KMSG_SECURITYLIST_OLD       = 0x0450 // 旧版证券列表
	KMSG_QUOTESLIST             = 0x054b // 排序行情列表
	KMSG_QUOTES                 = 0x054c // 批量行情
	KMSG_SECURITYQUOTES         = 0x053e // 行情信息
	KMSG_TOPBOARD               = 0x053f // 排行榜
	KMSG_UNUSUAL                = 0x0563 // 主力监控
	KMSG_AUCTION                = 0x056a // 集合竞价
	KMSG_CHARTSAMPLING          = 0x0fd1 // 抽样图
	KMSG_HISTORYORDERS          = 0x0fb4 // 历史委托
	KMSG_TRANSACTIONDATA        = 0x0fc5 // 分笔成交信息
	KMSG_XDXRINFO               = 0x000f // 除权除息信息
	KMSG_EXLOGIN                = 0x2454 // 扩展市场登录
	KMSG_EXSERVERINFO           = 0x2455 // 扩展市场服务信息
	KMSG_EXCOUNT                = 0x23f0 // 扩展市场数量
	KMSG_EXCATEGORYLIST         = 0x23f4 // 扩展市场分类列表
	KMSG_EXLIST                 = 0x23f5 // 扩展市场商品列表
	KMSG_EXKLINE                = 0x23ff // 扩展市场K线
	KMSG_EXHISTORYTRANSACTION   = 0x2412 // 扩展市场历史成交
	KMSG_EXTABLE                = 0x2422 // 扩展市场表格
	KMSG_EXTABLEDETAIL          = 0x2423 // 扩展市场详细表格
	KMSG_EXFILEMETA             = 0x2458 // 扩展市场文件元信息
	KMSG_EXFILEDOWNLOAD         = 0x2459 // 扩展市场文件下载
	KMSG_EXQUOTESLIST           = 0x2484 // 扩展市场行情列表
	KMSG_EXQUOTESINGLE          = 0x23fa // 扩展市场单个行情
	KMSG_EXQUOTES               = 0x248a // 扩展市场批量行情
	KMSG_EXQUOTES2              = 0x23fb // 扩展市场批量行情2
	KMSG_EXTICKCHART            = 0x248b // 扩展市场分时图
	KMSG_EXHISTORYTICKCHART     = 0x248c // 扩展市场历史分时图
	KMSG_EXCHARTSAMPLING        = 0x254d // 扩展市场抽样图
	KMSG_EXBOARDLIST            = 0x1231 // 扩展市场板块榜单

)

const (
	KLINE_TYPE_5MIN      = 0  // 5 分钟K 线
	KLINE_TYPE_15MIN     = 1  // 15 分钟K 线
	KLINE_TYPE_30MIN     = 2  // 30 分钟K 线
	KLINE_TYPE_1HOUR     = 3  // 1 小时K 线
	KLINE_TYPE_DAILY     = 4  // 日K 线
	KLINE_TYPE_WEEKLY    = 5  // 周K 线
	KLINE_TYPE_MONTHLY   = 6  // 月K 线
	KLINE_TYPE_EXHQ_1MIN = 7  // 1 分钟
	KLINE_TYPE_1MIN      = 8  // 1 分钟K 线
	KLINE_TYPE_RI_K      = 9  // 日K 线
	KLINE_TYPE_3MONTH    = 10 // 季K 线
	KLINE_TYPE_YEARLY    = 11 // 年K 线
)

type Msg interface {
	Serialize() ([]byte, error)
	UnSerialize(head interface{}, in []byte) error
}

var _seqId uint32

/*
0c 02000000 00 1c00 1c00 2d05 0100363030303030080001000000140000000000000000000000
0c 02189300 01 0300 0300 0d00 01
0c 00000000 00 0200 0200 1500
*/
type ReqHeader struct {
	Zip        uint8  // ZipFlag
	SeqID      uint32 // 请求编号
	PacketType uint8
	PkgLen1    uint16
	PkgLen2    uint16
	Method     uint16 // method 请求方法
}

type RespHeader struct {
	I1        uint32
	I2        uint8
	SeqID     uint32 // 请求编号
	I3        uint8
	Method    uint16 // method
	ZipSize   uint16 // 长度
	UnZipSize uint16 // 未压缩长度
}

func seqID() uint32 {
	atomic.AddUint32(&_seqId, 1)
	return _seqId
}

func todayDate() uint32 {
	now := time.Now()
	return uint32(now.Year()*10000 + int(now.Month())*100 + now.Day())
}

// pytdx : 类似utf-8的编码方式保存有符号数字
func getprice(b []byte, pos *int) int {
	/*
		    0x7f与常量做与运算实质是保留常量（转换为二进制形式）的后7位数，既取值区间为[0,127]
		    0x3f与常量做与运算实质是保留常量（转换为二进制形式）的后6位数，既取值区间为[0,63]

			0x80 1000 0000
			0x7f 0111 1111
			0x40  100 0000
			0x3f  011 1111
	*/
	posByte := 6
	bData := b[*pos]
	data := int(bData & 0x3f)
	bSign := false
	if (bData & 0x40) > 0 {
		bSign = true
	}

	if (bData & 0x80) > 0 {
		for {
			*pos += 1
			bData = b[*pos]
			data += (int(bData&0x7f) << posByte)

			posByte += 7

			if (bData & 0x80) <= 0 {
				break
			}
		}
	}
	*pos++

	if bSign {
		data = -data
	}
	return data
}

func gettime(b []byte, pos *int) (h uint16, m uint16) {
	var sec uint16
	binary.Read(bytes.NewBuffer(b[*pos:*pos+2]), binary.LittleEndian, &sec)
	h = sec / 60
	m = sec % 60
	(*pos) += 2
	return
}

func getfloat32(b []byte, pos *int) float64 {
	value := math.Float32frombits(binary.LittleEndian.Uint32(b[*pos : *pos+4]))
	(*pos) += 4
	return float64(value)
}

func decodeDateNum(category uint16, num uint32) (time.Time, bool) {
	minuteCategory := category < 4 || category == 7 || category == 8
	year, month, day := 0, 0, 0
	hour, minute := 15, 0

	if minuteCategory {
		zipData := num & 0xFFFF
		year = int((zipData >> 11) + 2004)
		month = int((zipData & 0x7FF) / 100)
		day = int((zipData & 0x7FF) % 100)

		totalMinutes := int(num >> 16)
		hour = totalMinutes / 60
		minute = totalMinutes % 60
	} else {
		year = int(num / 10000)
		month = int((num % 10000) / 100)
		day = int(num % 100)
	}

	if year < 2004 || year > time.Now().Year()+1 {
		return time.Time{}, false
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, false
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return time.Time{}, false
	}

	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.Local)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day || t.Hour() != hour || t.Minute() != minute {
		return time.Time{}, false
	}

	return t, true
}

func formatServerTime(raw int) string {
	if raw == 0 || raw == 100 {
		return "00:00:00.000"
	}

	ts := raw
	minutesField := (ts / 10000) % 100
	if minutesField < 60 {
		hours := ts / 1000000
		minutes := minutesField
		secondsMillis := float64(ts%10000) * 60 / 10000.0
		seconds := int(secondsMillis)
		millis := int((secondsMillis - float64(seconds)) * 1000)
		return time.Date(0, 1, 1, hours, minutes, seconds, millis*int(time.Millisecond), time.Local).Format("15:04:05.000")
	}

	total := float64(ts%1000000) * 60 / 1000000.0
	hours := ts / 1000000
	minutes := int(total / 60)
	secondsMillis := total - float64(minutes*60)
	seconds := int(secondsMillis)
	millis := int((secondsMillis - float64(seconds)) * 1000)
	return time.Date(0, 1, 1, hours, minutes, seconds, millis*int(time.Millisecond), time.Local).Format("15:04:05.000")
}

func parseYMD(raw uint32) (time.Time, error) {
	year := int(raw / 10000)
	month := int((raw % 10000) / 100)
	day := int(raw % 100)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}

func getdatetime(category int, b []byte, pos *int) (year int, month int, day int, hour int, minute int) {
	hour = 15
	if category < 4 || category == 7 || category == 8 {
		var zipday, tminutes uint16
		binary.Read(bytes.NewBuffer(b[*pos:*pos+2]), binary.LittleEndian, &zipday)
		(*pos) += 2
		binary.Read(bytes.NewBuffer(b[*pos:*pos+2]), binary.LittleEndian, &tminutes)
		(*pos) += 2

		year = int((zipday >> 11) + 2004)
		month = int((zipday % 2048) / 100)
		day = int((zipday % 2048) % 100)
		hour = int(tminutes / 60)
		minute = int(tminutes % 60)
	} else {
		var zipday uint32
		binary.Read(bytes.NewBuffer(b[*pos:*pos+4]), binary.LittleEndian, &zipday)
		(*pos) += 4
		year = int(zipday / 10000)
		month = int((zipday % 10000) / 100)
		day = int(zipday % 100)
	}
	return
}

func getdatetimenow(category int, lasttime string) (year int, month int, day int, hour int, minute int) {
	utime, _ := time.Parse("2006-01-02 15:04:05", lasttime)
	switch category {
	case KLINE_TYPE_5MIN:
		utime = utime.Add(time.Minute * 5)
	case KLINE_TYPE_15MIN:
		utime = utime.Add(time.Minute * 15)
	case KLINE_TYPE_30MIN:
		utime = utime.Add(time.Minute * 30)
	case KLINE_TYPE_1HOUR:
		utime = utime.Add(time.Hour)
	case KLINE_TYPE_DAILY:
		utime = utime.AddDate(0, 0, 1)
	case KLINE_TYPE_WEEKLY:
		utime = utime.Add(time.Hour * 24 * 7)
	case KLINE_TYPE_MONTHLY:
		utime = utime.AddDate(0, 1, 0)
	case KLINE_TYPE_EXHQ_1MIN:
		utime = utime.Add(time.Minute)
	case KLINE_TYPE_1MIN:
		utime = utime.Add(time.Minute)
	case KLINE_TYPE_RI_K:
		utime = utime.AddDate(0, 0, 1)
	case KLINE_TYPE_3MONTH:
		utime = utime.AddDate(0, 3, 0)
	case KLINE_TYPE_YEARLY:
		utime = utime.AddDate(1, 0, 0)
	}

	if category < 4 || category == 7 || category == 8 {
		if (utime.Hour() >= 15 && utime.Minute() > 0) || (utime.Hour() > 15) {
			utime = utime.AddDate(0, 0, 1)
			utime = utime.Add(time.Minute * 30)
			hour = (utime.Hour() + 18) % 24
		} else {
			hour = utime.Hour()
		}
		minute = utime.Minute()
	} else {
		if utime.Unix() > time.Now().Unix() {
			utime = time.Now()
		}
		hour = utime.Hour()
		minute = utime.Minute()
		if utime.Hour() > 15 {
			hour = 15
			minute = 0
		}
	}
	year = utime.Year()
	month = int(utime.Month())
	day = utime.Day()
	return
}

func getvolume(ivol int) (volume float64) {
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

func baseUnit(code string) float64 {
	switch code[:2] {
	case "60", "30", "68", "00":
		return 100.0
	default:
		return 1000.0
	}
}

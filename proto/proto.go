package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"sync/atomic"
	"time"

	"github.com/bensema/gotdx/types"
)

// MessageHeaderBytes 与 MessageMaxBytes 约定协议报文的头部长度与单条报文大小上限。
//
// MessageHeaderBytes 为请求与响应报文头固定占用的字节数；MessageMaxBytes 为单条报文
// 允许的最大字节数，超出该长度即视为非法报文。
const (
	MessageHeaderBytes = 0x10    // 请求与响应报文头固定占用的字节数，共十六字节。
	MessageMaxBytes    = 1 << 15 // 单条报文允许的最大字节数，共三万两千七百六十八字节，超出即视为非法报文。
)

// KMSG_* 是协议命令码常量集合，写入报文头的命令码字段以标识本次报文对应的业务协议；
// 服务端响应会回显同一命令码。
//
// 命令码按业务区段划分：`0x00xx` 为连接、公告与基础信息，`0x04xx`、`0x05xx` 为主站行情，
// `0x0fxx` 为历史数据，`0x12xx` 为 MAC 行情，`0x23xx`、`0x24xx` 为扩展市场，`0x26xx` 为客户端上报。
const (
	KMSG_EXCHANGEANNOUNCE       = 0x0002 // 交易所公告命令，对应 `ExchangeAnnouncement` 协议
	KMSG_CMD1                   = 0x000d // 首次握手命令，对应 `Hello1` 协议
	KMSG_CMD2                   = 0x0fdb // 第二次握手命令，对应 `Hello2` 协议
	KMSG_ANNOUNCEMENT           = 0x000a // 服务商公告命令，对应 `Announcement` 协议
	KMSG_TODOB                  = 0x000b // 未知协议命令，语义待确认
	KMSG_PING                   = 0x0015 // 服务端信息探测命令，对应 `Info` 协议
	KMSG_HEARTBEAT              = 0x0004 // 心跳保活命令，对应 `HeartBeat` 协议
	KMSG_SECURITYCOUNT          = 0x044e // 证券数量
	KMSG_BLOCKINFOMETA          = 0x02c5 // 板块文件信息
	KMSG_BLOCKINFO              = 0x06b9 // 板块文件
	KMSG_COMPANYCATEGORY        = 0x02cf // 公司信息文件信息
	KMSG_COMPANYCONTENT         = 0x02d0 // 公司信息描述
	KMSG_FINANCEINFO            = 0x0010 // 财务信息
	KMSG_HISTORYMINUTETIMEDATE  = 0x0feb // 历史分时信息
	KMSG_HISTORYTRANSACTIONDATA = 0x0fb5 // 历史分笔成交信息
	KMSG_INDEXBARS              = 0x0523 // 指数 K 线
	KMSG_VOLUMEPROFILE          = 0x051a // 成交分布
	KMSG_INDEXMOMENTUM          = 0x051c // 指数动量
	KMSG_INDEXINFO              = 0x051d // 指数概况
	KMSG_SECURITYBARS           = 0x0523 // 股票 K 线
	KMSG_SECURITYBARS_OFFSET    = 0x052d // 偏移 K 线
	KMSG_MINUTETIMEDATA         = 0x0537 // 分时数据
	KMSG_SECURITYLIST           = 0x044d // 证券列表
	KMSG_SECURITYLIST_OLD       = 0x0450 // 旧版证券列表
	KMSG_SECURITYFEATURE452     = 0x0452 // 证券扩展信息
	KMSG_QUOTESLIST             = 0x054b // 排序行情列表
	KMSG_QUOTES                 = 0x054c // 批量行情
	KMSG_QUOTESENCRYPT          = 0x0547 // 加密行情
	KMSG_SECURITYQUOTES         = 0x053e // 行情信息
	KMSG_TOPBOARD               = 0x053f // 排行榜
	KMSG_UNUSUAL                = 0x0563 // 主力监控
	KMSG_AUCTION                = 0x056a // 集合竞价
	KMSG_CHARTSAMPLING          = 0x0fd1 // 抽样图
	KMSG_HISTORYORDERS          = 0x0fb4 // 历史委托
	KMSG_TRANSACTIONDATA        = 0x0fc5 // 分笔成交信息
	KMSG_TRANSACTIONDATA_TRANS  = 0x0fc6 // 分笔成交信息（带方向）
	KMSG_TODOFDE                = 0x0fde // 未知协议，语义待确认
	KMSG_XDXRINFO               = 0x000f // 除权除息信息
	KMSG_EXLOGIN                = 0x2454 // 扩展市场登录
	KMSG_EXSERVERINFO           = 0x2455 // 扩展市场服务信息
	KMSG_EXCOUNT                = 0x23f0 // 扩展市场数量
	KMSG_EXCATEGORYLIST         = 0x23f4 // 扩展市场分类列表
	KMSG_EXLIST                 = 0x23f5 // 扩展市场标的列表
	KMSG_EXLIST_EXTRA           = 0x23f6 // 扩展市场试验列表
	KMSG_EXKLINE                = 0x23ff // 扩展市场 K 线
	KMSG_EXHISTORYTRANSACTION   = 0x2412 // 扩展市场历史成交
	KMSG_EXTABLE                = 0x2422 // 扩展市场表格
	KMSG_EXTABLEDETAIL          = 0x2423 // 扩展市场详细表格
	KMSG_EXFILEMETA             = 0x2458 // 扩展市场文件元信息
	KMSG_EXFILEDOWNLOAD         = 0x2459 // 扩展市场文件下载
	KMSG_EXQUOTESLIST           = 0x2484 // 扩展市场行情列表
	KMSG_EXQUOTES_EXPERIMENT1   = 0x2487 // 扩展市场试验报价一
	KMSG_EXQUOTES_EXPERIMENT2   = 0x2488 // 扩展市场试验报价二
	KMSG_EXKLINE2               = 0x2489 // 扩展市场 K 线二
	KMSG_EXQUOTESINGLE          = 0x23fa // 扩展市场单个行情
	KMSG_EXQUOTES               = 0x248a // 扩展市场批量行情
	KMSG_EXQUOTES2              = 0x23fb // 扩展市场批量行情二
	KMSG_EXTICKCHART            = 0x248b // 扩展市场分时图
	KMSG_EXHISTORYTICKCHART     = 0x248c // 扩展市场历史分时图
	KMSG_EXCHARTSAMPLING        = 0x254d // 扩展市场抽样图
	KMSG_EXMAPPING2562          = 0x2562 // 扩展市场映射信息
	KMSG_MACSERVERINFO          = 0x120f // MAC 服务端交易日时段
	KMSG_MACFILELIST            = 0x1215 // MAC 文件列表
	KMSG_MACFILEDOWNLOAD        = 0x1217 // MAC 文件下载
	KMSG_EXBOARDLIST            = 0x1231 // 扩展市场板块榜单
	KMSG_MACCAPITALFLOW         = 0x1218 // MAC 资金流向，头部标识为二
	KMSG_MACSYMBOLINFO          = 0x122a // MAC 股票摘要
	KMSG_MACSYMBOLQUOTES        = 0x122b // MAC 批量股票报价
	KMSG_MACTRANSACTIONS        = 0x122f // MAC 分时成交
	KMSG_MACMARKETMONITOR       = 0x1237 // MAC 市场监控
	KMSG_MACSYMBOLBELONGBOARD   = 0x1218 // MAC 股票所属板块
	KMSG_MACBOARDMEMBERS        = 0x122c // MAC 板块成员
	KMSG_MACQUOTES              = 0x122d // MAC 行情快照
	KMSG_MACSYMBOLBARS          = 0x122e // MAC 统一 K 线
	KMSG_MACAUCTION             = 0x123d // MAC 竞价
	KMSG_MACTICKCHARTS          = 0x123e // MAC 多日分时
	KMSG_MACKLINEOFFSET         = 0x124a // MAC K 线偏移信息
	KMSG_CLIENT264B             = 0x264b // 客户端信息 264B
	KMSG_CLIENT26AC             = 0x26ac // 客户端信息 26AC
	KMSG_CLIENT26AD             = 0x26ad // 客户端信息 26AD
	KMSG_CLIENT26AE             = 0x26ae // 客户端信息 26AE
	KMSG_CLIENT26B1             = 0x26b1 // 客户端信息 26B1

)

// RequestBuilder 负责把一次协议调用的参数编码为线上请求字节流。
// 实现方在 BuildRequest 中完成报文头与业务负载的组装；编码失败时返回非空错误。
type RequestBuilder interface {
	BuildRequest() ([]byte, error)
}

// ResponseParser 负责把服务端返回的字节流解析为强类型响应对象。
// ParseResponse 接收响应头与响应负载，解码失败时返回非空错误；解析成功后结果保存在协议对象内部。
type ResponseParser interface {
	ParseResponse(head *RespHeader, in []byte) error
}

// Protocol 组合请求构造、响应解析与结果读取三项能力，是单个协议的完整处理接口。
// 泛型参数 T 为该协议响应结果的类型；三个方法由同一协议对象提供，对象内部状态非并发安全，
// 不应被多个协程并发重复发起请求与解析。
type Protocol[T any] interface {
	RequestBuilder
	ResponseParser
	Response() T
}

var seqIDCounter atomic.Uint32

// ReqHeader 是发往服务端的请求报文头，共占 16 字节，由各协议请求在组装报文时填充。
//
// 该结构体由协议内部统一创建并填充，调用方通常无需直接构造；两个长度字段互为冗余校验。
type ReqHeader struct {
	Zip        uint8  // 压缩标志位；零表示不压缩，0x0c 为客户端常用取值。
	SeqID      uint32 // 请求编号，四字节；全局自增，与响应报文回显编号一一对应。
	PacketType uint8  // 报文类型；一为普通请求类报文。
	PkgLen1    uint16 // 报文体长度冗余字段一，两字节，单位为字节。
	PkgLen2    uint16 // 报文体长度冗余字段二，取值与 PkgLen1 相同。
	Method     uint16 // 协议命令码，两字节，取值见 KMSG_* 常量。
}

type genericReqHeader struct {
	Head       uint8  // 报文首字节标识。
	Customize  uint32 // 自定义四字节字段，语义随特殊协议而定。
	PacketType uint8  // 报文类型。
	PkgLen1    uint16 // 报文体长度冗余字段一，两字节。
	PkgLen2    uint16 // 报文体长度冗余字段二，取值与 PkgLen1 相同。
}

// RespHeader 是服务端返回报文的响应头，解析，由响应解码流程在读取负载前先取出。
//
// 前三个字段为协议保留标识字节，语义待确认；命令码与请求编号由服务端回显；
// ZipSize 为压缩后长度，UnZipSize 为解压后长度。
type RespHeader struct {
	I1        uint32 // 响应保留标识，四字节，语义待确认。
	I2        uint8  // 响应保留标识，语义待确认。
	SeqID     uint32 // 回显的请求编号，四字节，与请求 SeqID 对应。
	I3        uint8  // 响应保留标识，语义待确认。
	Method    uint16 // 回显的协议命令码，两字节。
	ZipSize   uint16 // 压缩后负载长度，两字节，单位为字节。
	UnZipSize uint16 // 解压后负载长度，两字节，单位为字节。
}

// seqID 返回全局自增请求编号，无入参；原子计数保证并发下编号唯一且单调递增，
// 返回的编号供各协议请求头的 SeqID 字段填充。
func seqID() uint32 {
	return seqIDCounter.Add(1)
}

// buildGenericRequest 构造不套用标准请求头布局的特殊协议报文；
// 两个长度字段记录同一份报文体长度，作为线上冗余校验。
//
// 参数 `head` 为报文首字节标识，`customize` 为自定义四字节字段，`packetType` 为报文类型，
// `method` 为协议命令码，`payload` 为业务负载字节流；返回拼装完成的完整报文字节流，
// 任一步二进制编码失败时返回错误。
func buildGenericRequest(head uint8, customize uint32, packetType uint8, method uint16, payload []byte) ([]byte, error) {
	// 先把命令码与业务负载顺序拼成报文体。
	body := new(bytes.Buffer)
	if err := binary.Write(body, binary.LittleEndian, method); err != nil {
		return nil, err
	}
	if _, err := body.Write(payload); err != nil {
		return nil, err
	}

	// 两个长度字段记录同一报文体长度，按写入专用请求头。
	header := genericReqHeader{
		Head:       head,
		Customize:  customize,
		PacketType: packetType,
		PkgLen1:    uint16(body.Len()),
		PkgLen2:    uint16(body.Len()),
	}

	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	if _, err := buf.Write(body.Bytes()); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// todayDate 返回当前本地日期按年月日依次占两位、打包成的整数，无入参；
// 返回整数形式如 20250512 对应 2025 年 5 月 12 日。
func todayDate() uint32 {
	now := time.Now()
	return uint32(now.Year()*10000 + int(now.Month())*100 + now.Day())
}

// getprice 按类 UTF-8 的变长有符号整数编码解析价格数值，并把偏移量推进到编码之后。
//
// 参数 `b` 为响应负载字节流，`pos` 为当前读取偏移，解码成功后向后推进到编码末尾；
// 首字节低 6 位为数据位、0x40 为符号位、0x80 表示后续仍有字节，后续字节低 7 位
// 按 7 位步长拼接；返回还原后的整数价格，符号位置位时结果取负。
func getprice(b []byte, pos *int) int {
	posByte := 6
	bData := b[*pos]
	// 首字节低六位提取数据位。
	data := int(bData & 0x3f)
	bSign := false
	// 0x40 符号位置位表示结果为负数。
	if (bData & 0x40) > 0 {
		bSign = true
	}

	// 0x80 续接标志置位时继续读取后续字节，低七位按七位步长并入结果。
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
	// 编码结束，偏移推进到下一字节。
	*pos++

	if bSign {
		data = -data
	}
	return data
}

// gettime 从响应字节流当前偏移读取当日累计分钟数，换算为时与分，并把偏移量推进两个字节。
//
// 参数 `b` 为响应负载字节流，`pos` 为当前读取偏移，读取后推进两字节；返回换算得到的时与分。
func gettime(b []byte, pos *int) (h uint16, m uint16) {
	var sec uint16
	binary.Read(bytes.NewBuffer(b[*pos:*pos+2]), binary.LittleEndian, &sec)
	h = sec / 60
	m = sec % 60
	(*pos) += 2
	return
}

// getfloat32 读取四字节单精度浮点数（成交量或成交额），扩展为双精度返回，并把偏移量推进四字节。
//
// 参数 `b` 为响应负载字节流，`pos` 为当前读取偏移，读取后推进四字节；返回提升为双精度的浮点数值。
func getfloat32(b []byte, pos *int) float64 {
	value := math.Float32frombits(binary.LittleEndian.Uint32(b[*pos : *pos+4]))
	(*pos) += 4
	return float64(value)
}

// DecodeSecond 把自 2004-01-01 本地纪元起经过的秒数换算为本地时间，用于秒级周期行情时间戳。
//
// 参数 `num` 为服务端给出的相对纪元秒数，不直接传递年月日；返回换算后的本地时间。
func DecodeSecond(num uint32) time.Time {
	epoch := time.Date(2004, 1, 0, 0, 0, 0, 0, time.Local)
	return epoch.Add(time.Duration(num) * time.Second)
}

// decodeDateNum 按周期类别解码行情日期编号，返回本地时间与是否解码成功。
//
// 参数 `category` 为周期类别，`num` 为服务端日期编号，`index` 为是否启用索引越界校验；
// 返回解码后的本地时间与成功标志，校验失败时返回零值时间与 `false`。
// 分钟周期与日及以上周期使用不同编码格式；启用索引校验时会拒绝越界的年月日与时分，
// 避免把脏数据写入时间对象。
func decodeDateNum(category uint16, num uint32, index bool) (time.Time, bool) {
	minuteCategory := category < 4 || category == 7 || category == 8
	isSecond := category == 13
	year, month, day := 0, 0, 0
	// 非分钟周期没有时分信息，默认取收盘时点。
	hour, minute, second := 15, 0, 0

	if minuteCategory {
		// 分钟周期：低十六位为位压缩日期，高十六位为当日累计分钟。
		zipData := num & 0xFFFF
		year = int((zipData >> 11) + 2004)
		month = int((zipData & 0x7FF) / 100)
		day = int((zipData & 0x7FF) % 100)

		totalMinutes := int(num >> 16)
		hour = totalMinutes / 60
		minute = totalMinutes % 60
	} else {
		// 日及以上周期：整数按年月日依次占两位打包。
		year = int(num / 10000)
		month = int((num % 10000) / 100)
		day = int(num % 100)
	}

	if isSecond {
		return DecodeSecond(num), true
	}
	if index {
		// if year == 2870 {
		// fmt.Println("----", num)
		// }

		// fmt.Println(year, month, day)

		// 历史脏数据曾出现离谱年份，先卡年份区间再解析月日。
		if year < 1992 || year > time.Now().Year()+1 {
			// fmt.Println(1)
			return time.Time{}, false
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			// fmt.Println(2)
			return time.Time{}, false
		}
		if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
			// fmt.Println(3)
			return time.Time{}, false
		}
	}

	t := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)

	return t, true
}

// formatServerTime 将服务端打包的时间整数格式化为「时:分:秒.毫秒」字符串。
//
// 参数 `raw` 为服务端打包的时间整数；返回格式化后的时间字符串。
// 服务端低位段以六十分之一档折算秒；分钟位本身不合法时说明已发生进位，需先把低位段整体折算为分钟再拆分。
// 传入值为 0 或 100 表示无有效时间，直接返回零时刻。
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

	// 分钟位超过 59 属于进位后的非法取值，按整段分钟重新拆分。
	total := float64(ts%1000000) * 60 / 1000000.0
	hours := ts / 1000000
	minutes := int(total / 60)
	secondsMillis := total - float64(minutes*60)
	seconds := int(secondsMillis)
	millis := int((secondsMillis - float64(seconds)) * 1000)
	return time.Date(0, 1, 1, hours, minutes, seconds, millis*int(time.Millisecond), time.Local).Format("15:04:05.000")
}

// parseYMD 把按年月日依次占两位打包的整数解析为本地日期，时分秒置零。
//
// 参数 `raw` 为打包日期整数；返回本地日期时间，当前实现不返回错误。
func parseYMD(raw uint32) (time.Time, error) {
	year := int(raw / 10000)
	month := int((raw % 10000) / 100)
	day := int(raw % 100)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local), nil
}

// getdatetime 从响应字节流当前偏移按周期类别读出年月日时分，编码格式与 `decodeDateNum` 对应。
//
// 参数 `category` 为周期类别，`b` 为响应负载字节流，`pos` 为当前读取偏移并随读取推进；
// 返回解析出的年、月、日、时、分。分钟周期依次读取位压缩日期与当日累计分钟；
// 日及以上周期读取打包日期；读取后偏移量随字节数推进。
func getdatetime(category int, b []byte, pos *int) (year int, month int, day int, hour int, minute int) {
	hour = 15
	if category < 4 || category == 7 || category == 8 {
		var zipday, tminutes uint16
		// 分钟周期：先读两字节位压缩日期，再读两字节当日累计分钟。
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
		// 日及以上周期：读取四字节打包日期。
		binary.Read(bytes.NewBuffer(b[*pos:*pos+4]), binary.LittleEndian, &zipday)
		(*pos) += 4
		year = int(zipday / 10000)
		month = int((zipday % 10000) / 100)
		day = int(zipday % 100)
	}
	return
}

// getdatetimenow 已知上一根 K 线时间，按周期类别向后推算下一根 K 线的时间。
//
// 参数 `category` 为周期类别，`lasttime` 为上一根 K 线已格式化的时间字符串；
// 返回推算出的下一根 K 线年、月、日、时、分。
// 分钟周期在收盘后需要跨日并叠加午休偏移；日及以上周期若推算结果落到未来，则回退为当前时间，
// 并把超过收盘时点的时刻截断为收盘时点。
func getdatetimenow(category int, lasttime string) (year int, month int, day int, hour int, minute int) {
	utime, _ := time.Parse("2006-01-02 15:04:05", lasttime)
	switch category {
	case types.KLINE_TYPE_5MIN:
		utime = utime.Add(time.Minute * 5)
	case types.KLINE_TYPE_15MIN:
		utime = utime.Add(time.Minute * 15)
	case types.KLINE_TYPE_30MIN:
		utime = utime.Add(time.Minute * 30)
	case types.KLINE_TYPE_1HOUR:
		utime = utime.Add(time.Hour)
	case types.KLINE_TYPE_DAILY:
		utime = utime.AddDate(0, 0, 1)
	case types.KLINE_TYPE_WEEKLY:
		utime = utime.Add(time.Hour * 24 * 7)
	case types.KLINE_TYPE_MONTHLY:
		utime = utime.AddDate(0, 1, 0)
	case types.KLINE_TYPE_EXHQ_1MIN:
		utime = utime.Add(time.Minute)
	case types.KLINE_TYPE_1MIN:
		utime = utime.Add(time.Minute)
	case types.KLINE_TYPE_RI_K:
		utime = utime.AddDate(0, 0, 1)
	case types.KLINE_TYPE_3MONTH:
		utime = utime.AddDate(0, 3, 0)
	case types.KLINE_TYPE_YEARLY:
		utime = utime.AddDate(1, 0, 0)
	}

	if category < 4 || category == 7 || category == 8 {
		// 已过收盘时点则跨到下一交易日，并叠加午休偏移；小时数按一天二十四小时回绕。
		if (utime.Hour() >= 15 && utime.Minute() > 0) || (utime.Hour() > 15) {
			utime = utime.AddDate(0, 0, 1)
			utime = utime.Add(time.Minute * 30)
			hour = (utime.Hour() + 18) % 24
		} else {
			hour = utime.Hour()
		}
		minute = utime.Minute()
	} else {
		// 推算结果晚于当前时刻意味着尚未形成该根 K 线，回退到当前时间。
		if utime.Unix() > time.Now().Unix() {
			utime = time.Now()
		}
		hour = utime.Hour()
		minute = utime.Minute()
		// 非分钟周期没有盘中时分，超出收盘时点一律截断为收盘时点。
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

// getvolume 解析服务端压缩传输的成交量：四字节拆出一个指数字节与三个尾数字节，
// 各段以二的幂为权重分别重建后求和；中段尾数最高位置位时，低位两段按双倍权重参与累加。
//
// 参数 `ivol` 为服务端四字节压缩编码的成交量整数；返回还原后的浮点成交量。
// 该算法由历史 C 实现平移移植，指数偏移常量沿用原实现，不便于直接按常规浮点编码理解。
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

// baseUnit 按证券代码前缀返回分时协议原始价格的缩放单位。
//
// 参数 `code` 为证券代码字符串；返回价格缩放分母，普通股票与指数为一百（两位小数），
// 上交所与深交所 ETF 为一千（三位小数）。
func baseUnit(code string) float64 {
	switch code[:2] {
	case "60", "30", "68", "00", "39":
		return 100.0
	case "51", "56", "58":
		// 上交所 ETF 使用三位小数。
		return 1000.0
	case "15":
		// 深交所 ETF 使用三位小数。
		return 1000.0
	default:
		return 100.0
	}
}

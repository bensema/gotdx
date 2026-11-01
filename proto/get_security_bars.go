package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
)

// GetSecurityBars 封装获取证券 K 线（含偏移量形式）的完整请求与响应处理流程。
//
// 该对象面向单次查询：先经 NewGetSecurityBars 构造，再依次调用 BuildRequest 编码、
// `ParseResponse` 解码、`Response` 读取结果；对象内部保存流程状态，非并发安全，
// 不应在多个协程间共享，零值不可直接使用。
type GetSecurityBars struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetSecurityBarsRequest
	reply      *GetSecurityBarsReply
}

// GetSecurityBarsRequest 是查询 K 线的请求参数，字段含义与 K 线查询协议一一对应。
type GetSecurityBarsRequest struct {
	Market   uint16  // 市场代码，取值见 `types` 包市场常量。
	Code     [6]byte // 六位证券代码，长度不足时由上层补零对齐。
	Category uint16  // K 线周期类别，取值为 `types.KLINE_TYPE_*` 常量。
	Times    uint16  // 周期倍数，零值按一处理；周线等长周期以倍数表达跨度。
	Start    uint16  // 起始偏移条数，从零开始计数。
	Count    uint16  // 请求 K 线条数；内部会再加一条用于推算首根昨收。
	Adjust   uint16  // 复权方式，取值为 `types.AdjustNone`、`types.AdjustQFQ`、`types.AdjustHFQ`。
	Reserved [8]byte // 协议保留填充区，必须填零。
}

// GetSecurityBarsReply 是查询 K 线的响应数据。
type GetSecurityBarsReply struct {
	Count uint16        // 实际返回条数，已扣除用于推算昨收的多取首条。
	List  []SecurityBar // 解析出的 K 线序列，无数据时为空切片。
}

// SecurityBar 表示一根 K 线，包含开高低收、成交量额、涨跌与时间等信息。
type SecurityBar struct {
	PreClose  float64   // 昨收价，单位为元，由上一根收盘价滚动推算。
	Open      float64   // 开盘价，单位为元；线上整数按千分比压缩（×1000），此处已还原为浮点。
	Close     float64   // 收盘价，单位为元；线上整数按千分比压缩（×1000），此处已还原为浮点。
	High      float64   // 最高价，单位为元；线上整数按千分比压缩（×1000），此处已还原为浮点。
	Low       float64   // 最低价，单位为元；线上整数按千分比压缩（×1000），此处已还原为浮点。
	Vol       float64   // 成交量，服务端以单精度浮点直接传输。
	Amount    float64   // 成交额，服务端以单精度浮点直接传输。
	Turnover  float64   // 换手率，由高层接口按尽力而为口径补齐。
	RisePrice float64   // 涨跌价，单位为元，收盘价相对昨收价的差值。
	RiseRate  float64   // 涨跌幅，百分比数值，涨跌价相对昨收价的比值。
	Year      int       // 年。
	Month     int       // 月，取值一至十二。
	Day       int       // 日，取值一至三十一。
	Hour      int       // 时，取值零至二十三。
	Minute    int       // 分，取值零至五十九。
	DateTime  time.Time // 组合后的本地时间。
	UpCount   uint16    // 上涨家数，指数类 K 线常见，个股协议下为零。
	DownCount uint16    // 下跌家数，指数类 K 线常见，个股协议下为零。
}

// NewGetSecurityBars 构造并初始化 K 线查询请求对象，参数 `req` 为非空的预填请求参数。
//
// 参数 `req` 中 `Market` 指定市场代码，`Code` 为六位证券代码，`Category` 为周期类别，
// `Times` 为周期倍数（零值按一处理），`Start` 为起始偏移条数，`Count` 为请求条数，
// `Adjust` 为复权方式。
//
// 实现中会额外多请求一条 K 线用于反推首根的昨收基准，因此实际请求条数为
// 入参 `Count` 加一，解析完成后响应条数会自动减回。
func NewGetSecurityBars(req *GetSecurityBarsRequest) *GetSecurityBars {
	// 多取一条 K 线，用于以首根收盘价推算后续涨跌价与涨跌幅。
	req.Count = req.Count + 1
	obj := new(GetSecurityBars)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetSecurityBarsRequest)
	obj.reply = new(GetSecurityBarsReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_SECURITYBARS

	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetSecurityBars) applyRequest(req *GetSecurityBarsRequest) {
	if req.Times == 0 {
		req.Times = 1
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 返回的字节流可直接写入连接；编码失败时返回错误，正常路径不返回错误。
func (obj *GetSecurityBars) BuildRequest() ([]byte, error) {
	// 报文体固定长度二十八字节，PkgLen1 与 PkgLen2 互为冗余校验。
	obj.reqHeader.PkgLen1 = 0x1c
	obj.reqHeader.PkgLen2 = 0x1c

	buf := new(bytes.Buffer)
	// 请求头与请求参数统一连续写入。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的 K 线数据并填充响应对象。
//
// 参数 `header` 为服务端响应头，`data` 为响应负载字节流。
// 数据布局：前 2 字节为条数；每条依次为 4 字节日期、开/收/高/低 4 个价格
// （变长整数编码，实际值 ×1000）、成交量与成交额两个 `float32`；
// 末尾可能附带指数类 K 线的上涨/下跌家数（见下方探测逻辑）。
// 报文长度不足或日期非法时返回错误。
func (obj *GetSecurityBars) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	isOffsetReq := obj.respHeader.Method == KMSG_SECURITYBARS_OFFSET

	pos := 0
	// 前 2 字节为返回的 K 线条数（含请求时多要的一条，用于反推首根涨跌）。
	if err := binary.Read(bytes.NewBuffer(data[pos:pos+2]), binary.LittleEndian, &obj.reply.Count); err != nil {
		return err
	}
	pos += 2

	var preCloseRaw int // 昨收盘价，随遍历滚动更新。
	for index := uint16(0); index < obj.reply.Count; index++ {
		var dateNum uint32
		if err := binary.Read(bytes.NewBuffer(data[pos:pos+4]), binary.LittleEndian, &dateNum); err != nil {
			return err
		}
		pos += 4

		// 日期按 K 线周期类别解码（日线用年月日、分时/秒线用时间偏移等）。
		dateTime, ok := decodeDateNum(obj.request.Category, dateNum, false)
		if !ok {
			return fmt.Errorf("invalid kline datetime: %d", dateNum)
		}

		var openRaw, closeRaw, highRaw, lowRaw int

		openRaw = getprice(data, &pos)
		if isOffsetReq {
			// 偏移 K 线协议只返回相对基准的增量价格，须累加昨收还原绝对价。
			openRaw = preCloseRaw + openRaw
			closeRaw = getprice(data, &pos) + openRaw
			highRaw = getprice(data, &pos) + openRaw
			lowRaw = getprice(data, &pos) + openRaw
		} else {
			closeRaw = getprice(data, &pos)
			highRaw = getprice(data, &pos)
			lowRaw = getprice(data, &pos)
		}

		vol := getfloat32(data, &pos)
		amount := getfloat32(data, &pos)

		bar := SecurityBar{
			// 价格以 ×1000 的整数压缩存储（`getprice` 变长解码），此处转回浮点。
			Open:     float64(openRaw) / 1000.0,
			Close:    float64(closeRaw) / 1000.0,
			High:     float64(highRaw) / 1000.0,
			Low:      float64(lowRaw) / 1000.0,
			Vol:      vol,
			Amount:   amount,
			Year:     dateTime.Year(),
			Month:    int(dateTime.Month()),
			Day:      dateTime.Day(),
			Hour:     dateTime.Hour(),
			Minute:   dateTime.Minute(),
			DateTime: dateTime,
		}
		bar.PreClose = float64(preCloseRaw) / 1000.0
		preCloseRaw = closeRaw // 记录本根收盘价，作为下一根的昨收/增量基准。

		bar.RiseRate = bar.GetRiseRate()
		bar.RisePrice = bar.GetRisePrice()

		if pos+4 <= len(data) {
			// 探测末尾 4 字节：若无法按日期解码，则视为指数类 K 线的上涨/下跌家数。
			tryDateNum := binary.LittleEndian.Uint32(data[pos : pos+4])
			if _, ok := decodeDateNum(obj.request.Category, tryDateNum, false); !ok {
				bar.UpCount = binary.LittleEndian.Uint16(data[pos : pos+2])
				bar.DownCount = binary.LittleEndian.Uint16(data[pos+2 : pos+4])
				pos += 4
			}
		}
		// NewGetSecurityBars 多请求了一条用于推算昨收的数据，因此第一条予以舍弃。
		if index == 0 {
			continue
		}
		obj.reply.List = append(obj.reply.List, bar)
	}
	// 条数同样减一，与丢弃首条的逻辑保持一致。
	obj.reply.Count = obj.reply.Count - 1

	return nil
}

// GetRisePrice 计算涨跌价，即收盘价相对昨收价的差值。
//
// 昨收缺失（为 0）时回退为相对开盘价计算，避免除零与负基准失真。
func (bar SecurityBar) GetRisePrice() float64 {
	if bar.PreClose == 0 {
		// 昨收缺失时退化为相对开盘价，保证返回值可用。
		return bar.Close - bar.Open
	}
	return bar.Close - bar.PreClose
}

// GetRiseRate 计算涨跌幅，即涨跌价相对昨收价的百分比。
//
// 昨收缺失（为 0）时回退为相对开盘价的百分比，语义与 GetRisePrice 一致。
func (bar SecurityBar) GetRiseRate() float64 {
	if bar.PreClose == 0 {
		return float64(bar.Close-bar.Open) / float64(bar.Open) * 100
	}
	return float64(bar.Close-bar.PreClose) / float64(bar.PreClose) * 100
}

// Response 返回解析完成后的 K 线响应数据，未调用 ParseResponse 前内容为零值。
func (obj *GetSecurityBars) Response() *GetSecurityBarsReply {
	return obj.reply
}

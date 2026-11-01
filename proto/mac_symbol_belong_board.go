package proto

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
)

// MACSymbolBelongBoard 查询单只标的所属的板块及其行情信息。
//
// 使用时先由 NewMACSymbolBelongBoard 构造对象，再依次调用 BuildRequest、ParseResponse，最后读取响应结果。
// 该对象面向单次请求使用，内部未做并发同步，不应在多个并发协程间共享；零值不可直接使用。
type MACSymbolBelongBoard struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACSymbolBelongBoardRequest
	reply      *MACSymbolBelongBoardReply
}

// MACSymbolBelongBoardRequest 保存所属板块查询的市场、标的代码与查询标识参数。
type MACSymbolBelongBoardRequest struct {
	Market   uint16   // 市场代码，两字节，取值见 `types.Market*` 常量。
	Symbol   [8]byte  // 证券代码，GBK 编码定长 8 字节，不足补空字符。
	Reserved [16]byte // 协议保留填充字节，填零。
	Query    [21]byte // 查询标识文本，默认为 `Stock_GLHQ`（所属板块），GBK 编码定长 21 字节。
}

// MACSymbolBelongBoardReply 保存所属板块响应的市场、查询文本与逐条板块条目。
type MACSymbolBelongBoardReply struct {
	Market uint16               // 市场代码，两字节。
	Query  string               // 查询标识回显文本，GBK 解码。
	List   []MACBelongBoardItem // 标的所属板块条目列表。
}

// MACBelongBoardItem 表示标的所属的一个板块条目及其行情信息。
//
// 十三列布局额外携带板块代表标的行情；三个通用占位指标槽的具体含义随板块类型变化，
// 语义待确认。
type MACBelongBoardItem struct {
	BoardType     string  // 板块类型文本（JSON 首列）。
	MarketCode    int     // 市场代码（JSON 第二列）。
	StatusCode    int     // 复用市场代码列得到的状态码，口径语义待确认。
	BoardCode     string  // 板块代码（JSON 第三列）。
	BoardName     string  // 板块名称（JSON 第四列）。
	Price         float64 // 板块现价（JSON 第五列）。
	PreClose      float64 // 板块昨收价（JSON 第六列）。
	SchemaColumns int     // 本条 JSON 列数，用于区分九列与十三列两种布局。

	LimitUpCount   float64 // 涨停家数，仅九列布局填充。
	LimitDownCount float64 // 跌停家数，仅九列布局填充。
	MostSimilar    float64 // 相似度指标，仅九列布局填充，口径语义待确认。

	SpeedPct       float64 // 板块涨速，仅十三列布局填充。
	SymbolMarket   int     // 代表标的市场代码，仅十三列布局填充。
	Symbol         string  // 代表标的代码，仅十三列布局填充。
	SymbolName     string  // 代表标的名称，仅十三列布局填充。
	SymbolClose    float64 // 代表标的收盘价，仅十三列布局填充。
	SymbolPreClose float64 // 代表标的昨收价，仅十三列布局填充。
	SymbolSpeedPct float64 // 代表标的涨速，仅十三列布局填充。

	Metric1 float64 // 通用占位指标一，含义随布局（九列/十三列）变化，语义待确认。
	Metric2 float64 // 通用占位指标二，含义随布局变化，语义待确认。
	Metric3 float64 // 通用占位指标三，含义随布局变化，语义待确认。
}

// NewMACSymbolBelongBoard 创建所属板块查询对象；请求参数为空时使用内置默认查询标识。
//
// 参数 `req` 指定预填查询参数，可传 `nil`：`Market` 为市场代码、`Symbol` 为证券代码、
// `Query` 为查询标识；`Query` 为空时回填默认标识 `Stock_GLHQ`。
func NewMACSymbolBelongBoard(req *MACSymbolBelongBoardRequest) *MACSymbolBelongBoard {
	obj := &MACSymbolBelongBoard{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACSymbolBelongBoardRequest),
		reply:      new(MACSymbolBelongBoardReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACSYMBOLBELONGBOARD
	obj.request.Query = makeMACCode21("Stock_GLHQ")
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACSymbolBelongBoard) applyRequest(req *MACSymbolBelongBoardRequest) {
	if req.Query == ([21]byte{}) {
		req.Query = makeMACCode21("Stock_GLHQ")
	}
	obj.request = req
}

// BuildRequest 编码请求参数并返回完整协议报文；结构体序列化失败时返回错误。
//
// 该方法无外部入参，从已填充的 `request` 读取参数：整个请求结构整体写入负载，
// 再由 `buildExRequest` 追加命令码 `KMSG_MACSYMBOLBELONGBOARD` 与请求头封装成完整报文。
func (obj *MACSymbolBelongBoard) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求结构字段与线上布局一致，整体序列化即可。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	// 追加命令码与标准请求头，组装为可直接发送的完整报文。
	return buildExRequest(KMSG_MACSYMBOLBELONGBOARD, payload.Bytes())
}

// ParseResponse 解码服务端返回的字节流并填充响应；报文长度不足或负载不是合法 JSON 时返回错误。
//
// 参数 `header` 为服务端回传的响应头，`data` 为解压后的响应负载字节流。
func (obj *MACSymbolBelongBoard) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 27 {
		return fmt.Errorf("invalid mac belong board response length: %d", len(data))
	}

	// 头部：市场（0:2）、查询文本（2:14）；27 字节之后为 JSON 负载。
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Query = Utf8ToGbk(data[2:14])

	// JSON 元素数值类型不固定，先解为通用二维行集合，再按列类型逐项转换。
	var rows [][]interface{}
	if err := json.Unmarshal([]byte(Utf8ToGbk(data[27:])), &rows); err != nil {
		return err
	}

	for _, row := range rows {
		// 最少九列才能构成一条有效板块记录。
		if len(row) < 9 {
			continue
		}
		// 公共列：板块类型、市场代码、板块代码、板块名称、板块现价、昨收。
		item := MACBelongBoardItem{
			BoardType:     anyToString(row[0]),
			MarketCode:    anyToInt(row[1]),
			BoardCode:     anyToString(row[2]),
			BoardName:     anyToString(row[3]),
			Price:         anyToFloat64(row[4]),
			PreClose:      anyToFloat64(row[5]),
			SchemaColumns: len(row),
		}
		item.StatusCode = item.MarketCode

		// 按列宽分两种布局：九列为涨跌停与相似度统计，十三列追加代表标的行情。
		switch len(row) {
		case 9:
			// 九列：涨停数、跌停数、相似度，同时回填三个通用指标槽。
			item.LimitUpCount = anyToFloat64(row[6])
			item.LimitDownCount = anyToFloat64(row[7])
			item.MostSimilar = anyToFloat64(row[8])
			item.Metric1 = item.LimitUpCount
			item.Metric2 = item.LimitDownCount
			item.Metric3 = item.MostSimilar
		case 13:
			// 13 列：板块涨速及代表标的市场、代码、名称、收盘、昨收、涨速。
			item.SpeedPct = anyToFloat64(row[6])
			item.SymbolMarket = anyToInt(row[7])
			item.Symbol = anyToString(row[8])
			item.SymbolName = anyToString(row[9])
			item.SymbolClose = anyToFloat64(row[10])
			item.SymbolPreClose = anyToFloat64(row[11])
			item.SymbolSpeedPct = anyToFloat64(row[12])
			item.Metric1 = item.SpeedPct
			item.Metric2 = item.SymbolClose
			item.Metric3 = item.SymbolSpeedPct
		default:
			// 未知列宽仅按通用指标槽取后三列，不强行映射到具体业务字段。
			item.Metric1 = anyToFloat64(row[6])
			item.Metric2 = anyToFloat64(row[7])
			item.Metric3 = anyToFloat64(row[8])
		}
		obj.reply.List = append(obj.reply.List, item)
	}

	return nil
}

// Response 返回解码完成的所属板块响应；未调用 ParseResponse 前内容为零值。
func (obj *MACSymbolBelongBoard) Response() *MACSymbolBelongBoardReply {
	return obj.reply
}

// anyToString 兼容 JSON 字符串、数值与原始数字文本等形态，统一转换为字符串。
//
// 参数 `v` 为 JSON 解出的任意类型单元格值；返回值为对应字符串，无法识别形态交由默认格式化兜底。
func anyToString(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		return value.String()
	default:
		return fmt.Sprint(value)
	}
}

// anyToInt 将 JSON 数值或数字字符串兼容转换为整数，无法解析时返回 0。
//
// 参数 `v` 为 JSON 解出的任意类型单元格值；返回值为整数，解析失败取零值。
func anyToInt(v interface{}) int {
	switch value := v.(type) {
	case float64:
		return int(value)
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(value)
		return i
	default:
		return 0
	}
}

// anyToFloat64 将 JSON 数值或数字字符串兼容转换为浮点，无法解析时返回 0。
//
// 参数 `v` 为 JSON 解出的任意类型单元格值；返回值为浮点数值，解析失败取零值。
func anyToFloat64(v interface{}) float64 {
	switch value := v.(type) {
	case float64:
		return value
	case json.Number:
		f, _ := value.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(value, 64)
		return f
	default:
		return 0
	}
}

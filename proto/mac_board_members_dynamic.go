package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// MACBoardMembersQuotesDynamic 表示按位图动态解析的 MAC 板块成分报价协议。
type MACBoardMembersQuotesDynamic struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *MACBoardMembersQuotesDynamicRequest
	reply      *MACBoardMembersQuotesDynamicReply
}

// MACBoardMembersQuotesDynamicRequest 表示动态字段成分报价请求。
type MACBoardMembersQuotesDynamicRequest struct {
	BoardCode   uint32
	Reserved1   [9]byte
	SortType    uint16
	Start       uint32
	PageSize    uint8
	Zero        uint8
	SortOrder   uint8
	Filter      uint8
	FieldBitmap [20]byte
}

// MACDynamicFieldDef 描述一个由位图激活的动态字段。
type MACDynamicFieldDef struct {
	Bit         uint8
	Name        string
	Format      string
	Description string
}

// MACBoardMemberQuoteDynamicItem 表示单只成分股的动态字段结果。
type MACBoardMemberQuoteDynamicItem struct {
	Name   string
	Market uint16
	Symbol string
	Values map[string]any
}

// MACBoardMembersQuotesDynamicReply 表示动态字段成分报价响应。
type MACBoardMembersQuotesDynamicReply struct {
	FieldBitmap  [20]byte
	ActiveFields []MACDynamicFieldDef
	Count        uint16
	Total        uint32
	Stocks       []MACBoardMemberQuoteDynamicItem
}

var macBoardMembersQuotesDynamicFieldMap = map[uint8]MACDynamicFieldDef{
	0x0:  {Bit: 0x0, Name: "pre_close", Format: "float32", Description: "昨收盘价"},
	0x1:  {Bit: 0x1, Name: "open", Format: "float32", Description: "开盘价"},
	0x2:  {Bit: 0x2, Name: "high", Format: "float32", Description: "最高价"},
	0x3:  {Bit: 0x3, Name: "low", Format: "float32", Description: "最低价"},
	0x4:  {Bit: 0x4, Name: "close", Format: "float32", Description: "收盘价"},
	0x5:  {Bit: 0x5, Name: "vol", Format: "uint32", Description: "成交量"},
	0x6:  {Bit: 0x6, Name: "vol_ratio", Format: "float32", Description: "量比"},
	0x7:  {Bit: 0x7, Name: "amount", Format: "float32", Description: "总金额"},
	0x8:  {Bit: 0x8, Name: "inside_volume", Format: "uint32", Description: "内盘"},
	0x9:  {Bit: 0x9, Name: "outside_volume", Format: "uint32", Description: "外盘"},
	0xa:  {Bit: 0xa, Name: "total_shares", Format: "float32", Description: "总股数"},
	0xb:  {Bit: 0xb, Name: "total_shares_hk", Format: "float32", Description: "H 股数"},
	0xc:  {Bit: 0xc, Name: "eps", Format: "float32", Description: "每股收益"},
	0xd:  {Bit: 0xd, Name: "net_assets", Format: "float32", Description: "净资产"},
	0xe:  {Bit: 0xe, Name: "action_price", Format: "float32", Description: "未知价"},
	0xf:  {Bit: 0xf, Name: "total_market_cap_ab", Format: "float32", Description: "AB 股总市值"},
	0x10: {Bit: 0x10, Name: "pe_dynamic", Format: "float32", Description: "市盈率(动)"},
	0x11: {Bit: 0x11, Name: "bid", Format: "float32", Description: "买价"},
	0x12: {Bit: 0x12, Name: "ask", Format: "float32", Description: "卖价"},
	0x13: {Bit: 0x13, Name: "server_update_date", Format: "uint32", Description: "服务器更新日期"},
	0x14: {Bit: 0x14, Name: "server_update_time", Format: "uint32", Description: "服务器更新时间"},
	0x15: {Bit: 0x15, Name: "lot_size_info", Format: "uint32", Description: "每手信息"},
	0x16: {Bit: 0x16, Name: "unknown_22", Format: "float32", Description: "未知字段 22"},
	0x17: {Bit: 0x17, Name: "dividend_yield", Format: "float32", Description: "股息"},
	0x18: {Bit: 0x18, Name: "bid_volume", Format: "uint32", Description: "买量"},
	0x19: {Bit: 0x19, Name: "ask_volume", Format: "uint32", Description: "卖量"},
	0x1a: {Bit: 0x1a, Name: "last_volume", Format: "uint32", Description: "现量"},
	0x1b: {Bit: 0x1b, Name: "turnover", Format: "float32", Description: "换手"},
	0x1c: {Bit: 0x1c, Name: "block5", Format: "uint32", Description: "行业分类代码"},
	0x1d: {Bit: 0x1d, Name: "block_ext_info", Format: "uint32", Description: "行业扩展 ID"},
	0x1e: {Bit: 0x1e, Name: "some_bitmap", Format: "uint32", Description: "位图字段"},
	0x1f: {Bit: 0x1f, Name: "decimal_point", Format: "uint32", Description: "数据精度"},
	0x20: {Bit: 0x20, Name: "buy_price_limit", Format: "float32", Description: "涨停价"},
	0x21: {Bit: 0x21, Name: "sell_price_limit", Format: "float32", Description: "跌停价"},
	0x22: {Bit: 0x22, Name: "unknown_34", Format: "uint32", Description: "未知字段 34"},
	0x23: {Bit: 0x23, Name: "lot_size", Format: "uint32", Description: "每手股数"},
	0x24: {Bit: 0x24, Name: "float_shares", Format: "float32", Description: "流通股"},
	0x25: {Bit: 0x25, Name: "speed_pct", Format: "float32", Description: "涨速"},
	0x26: {Bit: 0x26, Name: "avg_price", Format: "float32", Description: "均价"},
	0x27: {Bit: 0x27, Name: "float_shares2", Format: "float32", Description: "流通股(备用)"},
	0x28: {Bit: 0x28, Name: "pe_ttm_vol_related", Format: "float32", Description: "市盈率 TTM 相关"},
	0x29: {Bit: 0x29, Name: "close_placeholder", Format: "float32", Description: "收盘价占位"},
	0x2a: {Bit: 0x2a, Name: "unknown_42", Format: "float32", Description: "未知字段 42"},
	0x2b: {Bit: 0x2b, Name: "kcb_flag", Format: "uint32", Description: "科创板标志"},
	0x2c: {Bit: 0x2c, Name: "bj_flag", Format: "uint32", Description: "北交所标志"},
	0x2d: {Bit: 0x2d, Name: "unknown_45", Format: "float32", Description: "未知字段 45"},
	0x2e: {Bit: 0x2e, Name: "unknown_46", Format: "float32", Description: "未知字段 46"},
	0x2f: {Bit: 0x2f, Name: "unknown_47", Format: "float32", Description: "未知字段 47"},
	0x30: {Bit: 0x30, Name: "pe_ttm", Format: "float32", Description: "市盈率 TTM"},
	0x31: {Bit: 0x31, Name: "pe_static", Format: "float32", Description: "市盈率静"},
	0x32: {Bit: 0x32, Name: "unknown_50", Format: "uint32", Description: "未知字段 50"},
	0x33: {Bit: 0x33, Name: "unknown_51", Format: "uint32", Description: "未知字段 51"},
	0x34: {Bit: 0x34, Name: "unknown_52", Format: "uint32", Description: "未知字段 52"},
	0x35: {Bit: 0x35, Name: "unknown_53", Format: "float32", Description: "未知字段 53"},
	0x36: {Bit: 0x36, Name: "unknown_54", Format: "float32", Description: "未知字段 54"},
	0x37: {Bit: 0x37, Name: "unknown_55", Format: "uint32", Description: "未知字段 55"},
	0x38: {Bit: 0x38, Name: "unknown_close_price", Format: "float32", Description: "美股字段"},
	0x39: {Bit: 0x39, Name: "unknown_57", Format: "float32", Description: "未知字段 57"},
	0x3a: {Bit: 0x3a, Name: "unknown_58", Format: "uint32", Description: "未知字段 58"},
	0x3b: {Bit: 0x3b, Name: "change_20d_pct", Format: "float32", Description: "20 日涨幅%"},
	0x3c: {Bit: 0x3c, Name: "ytd_pct", Format: "float32", Description: "年初至今%"},
	0x3d: {Bit: 0x3d, Name: "unknown_61", Format: "float32", Description: "未知字段 61"},
	0x3e: {Bit: 0x3e, Name: "unknown_62", Format: "float32", Description: "未知字段 62"},
	0x3f: {Bit: 0x3f, Name: "unknown_63", Format: "uint32", Description: "未知字段 63"},
	0x40: {Bit: 0x40, Name: "mtd_pct", Format: "float32", Description: "月初至今%"},
	0x41: {Bit: 0x41, Name: "change_1y_pct", Format: "float32", Description: "一年涨幅%"},
	0x42: {Bit: 0x42, Name: "prev_change_pct", Format: "float32", Description: "昨涨幅%"},
	0x43: {Bit: 0x43, Name: "change_3d_pct", Format: "float32", Description: "3 日涨幅%"},
	0x44: {Bit: 0x44, Name: "change_60d_pct", Format: "float32", Description: "60 日涨幅%"},
	0x45: {Bit: 0x45, Name: "change_5d_pct", Format: "float32", Description: "5 日涨幅%"},
	0x46: {Bit: 0x46, Name: "change_10d_pct", Format: "float32", Description: "10 日涨幅%"},
	0x47: {Bit: 0x47, Name: "unknown_71", Format: "float32", Description: "未知字段 71"},
	0x48: {Bit: 0x48, Name: "low_copy", Format: "float32", Description: "最低价备份"},
	0x49: {Bit: 0x49, Name: "low_copy2", Format: "float32", Description: "最低价备份 2"},
	0x4a: {Bit: 0x4a, Name: "ah_code", Format: "uint32", Description: "对应 A/H 股代码"},
	0x4b: {Bit: 0x4b, Name: "unknown_code", Format: "uint32", Description: "未知代码"},
	0x4c: {Bit: 0x4c, Name: "unknown_76", Format: "float32", Description: "未知字段 76"},
	0x4d: {Bit: 0x4d, Name: "unknown_77", Format: "float32", Description: "未知字段 77"},
	0x4e: {Bit: 0x4e, Name: "unknown_78", Format: "float32", Description: "未知字段 78"},
	0x4f: {Bit: 0x4f, Name: "unknown_79", Format: "float32", Description: "未知字段 79"},
	0x50: {Bit: 0x50, Name: "unknown_80", Format: "float32", Description: "未知字段 80"},
	0x51: {Bit: 0x51, Name: "unknown_81", Format: "float32", Description: "未知字段 81"},
	0x52: {Bit: 0x52, Name: "unknown_82", Format: "float32", Description: "未知字段 82"},
	0x53: {Bit: 0x53, Name: "unknown_83", Format: "float32", Description: "未知字段 83"},
	0x54: {Bit: 0x54, Name: "unknown_84", Format: "float32", Description: "未知字段 84"},
	0x55: {Bit: 0x55, Name: "unknown_85", Format: "float32", Description: "未知字段 85"},
	0x56: {Bit: 0x56, Name: "unknown_86", Format: "float32", Description: "未知字段 86"},
	0x57: {Bit: 0x57, Name: "open_amount", Format: "float32", Description: "开盘金额"},
}

func defaultMACBoardMembersQuotesFieldBitmap() [20]byte {
	return [20]byte{
		0xff, 0xfc, 0xe1, 0xcc, 0x3f, 0x08, 0x03, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
}

// NewMACBoardMembersQuotesDynamic 创建动态字段成分报价协议对象。
func NewMACBoardMembersQuotesDynamic(req *MACBoardMembersQuotesDynamicRequest) *MACBoardMembersQuotesDynamic {
	obj := &MACBoardMembersQuotesDynamic{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(MACBoardMembersQuotesDynamicRequest),
		reply:      new(MACBoardMembersQuotesDynamicReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_MACBOARDMEMBERS
	obj.request.SortType = 14
	obj.request.PageSize = 80
	obj.request.SortOrder = 1
	obj.request.FieldBitmap = defaultMACBoardMembersQuotesFieldBitmap()
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *MACBoardMembersQuotesDynamic) applyRequest(req *MACBoardMembersQuotesDynamicRequest) {
	if req.PageSize == 0 {
		req.PageSize = 80
	}
	if req.SortType == 0 {
		req.SortType = 14
	}
	if req.SortOrder == 0 {
		req.SortOrder = 1
	}
	if req.FieldBitmap == ([20]byte{}) {
		req.FieldBitmap = defaultMACBoardMembersQuotesFieldBitmap()
	}
	obj.request = req
}

func (obj *MACBoardMembersQuotesDynamic) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request.BoardCode); err != nil {
		return nil, err
	}
	if _, err := payload.Write(obj.request.Reserved1[:]); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.SortType); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Start); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.PageSize); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Zero); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.SortOrder); err != nil {
		return nil, err
	}
	if err := binary.Write(payload, binary.LittleEndian, obj.request.Filter); err != nil {
		return nil, err
	}
	if _, err := payload.Write(obj.request.FieldBitmap[:]); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_MACBOARDMEMBERS, payload.Bytes())
}

func (obj *MACBoardMembersQuotesDynamic) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	if len(data) < 26 {
		return fmt.Errorf("invalid mac board members dynamic response length: %d", len(data))
	}

	copy(obj.reply.FieldBitmap[:], data[:20])
	obj.reply.Total = binary.LittleEndian.Uint32(data[20:24])
	obj.reply.Count = binary.LittleEndian.Uint16(data[24:26])
	obj.reply.ActiveFields = activeMACDynamicFields(obj.reply.FieldBitmap)

	rowLength := 68 + len(obj.reply.ActiveFields)*4
	pos := 26
	for i := uint16(0); i < obj.reply.Count; i++ {
		if pos+rowLength > len(data) {
			return fmt.Errorf("invalid mac dynamic quote item %d", i)
		}
		row := data[pos : pos+rowLength]
		item := MACBoardMemberQuoteDynamicItem{
			Market: binary.LittleEndian.Uint16(row[:2]),
			Symbol: Utf8ToGbk(row[2:24]),
			Name:   Utf8ToGbk(row[24:68]),
			Values: make(map[string]any, len(obj.reply.ActiveFields)),
		}
		fieldPos := 68
		for _, field := range obj.reply.ActiveFields {
			raw := row[fieldPos : fieldPos+4]
			switch field.Format {
			case "uint32":
				item.Values[field.Name] = binary.LittleEndian.Uint32(raw)
			default:
				item.Values[field.Name] = float64(math.Float32frombits(binary.LittleEndian.Uint32(raw)))
			}
			fieldPos += 4
		}
		obj.reply.Stocks = append(obj.reply.Stocks, item)
		pos += rowLength
	}

	return nil
}

// Response 返回动态字段成分报价响应。
func (obj *MACBoardMembersQuotesDynamic) Response() *MACBoardMembersQuotesDynamicReply {
	return obj.reply
}

func activeMACDynamicFields(bitmap [20]byte) []MACDynamicFieldDef {
	fields := make([]MACDynamicFieldDef, 0)
	for bit := 0; bit < len(bitmap)*8; bit++ {
		if bitmap[bit/8]&(1<<uint(bit%8)) == 0 {
			continue
		}
		fieldDef, ok := macBoardMembersQuotesDynamicFieldMap[uint8(bit)]
		if !ok {
			fieldDef = MACDynamicFieldDef{
				Bit:         uint8(bit),
				Name:        fmt.Sprintf("unknown_field_%d", bit),
				Format:      "uint32",
				Description: "未映射字段",
			}
		}
		fields = append(fields, fieldDef)
	}
	return fields
}

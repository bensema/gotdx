package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
	"strings"
	"time"
)

// GetCompanyCategory 封装上市公司 F10 资料目录分类查询的完整请求与响应处理流程。
//
// F10 资料分两步获取：先用本协议查询某只证券拥有哪些资料分类（名称、文件名、文件内
// 偏移与长度），再由 GetCompanyContent 按文件名与偏移读取对应正文。
type GetCompanyCategory struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetCompanyCategoryRequest
	reply      *GetCompanyCategoryReply
}

// GetCompanyCategoryRequest 是公司资料目录查询的请求参数。
type GetCompanyCategoryRequest struct {
	Market uint16  // 市场代码（2 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
	Zero   uint32  // 协议保留占位，发送时填 0，语义待确认。
}

// GetCompanyCategoryReply 是公司资料目录查询的响应数据。
type GetCompanyCategoryReply struct {
	Count      uint16            // 服务端返回的目录条目条数。
	Categories []CompanyCategory // 解析后的资料分类列表。
}

// CompanyCategory 表示一个公司资料分类条目，包含名称、对应文件名及文件内偏移与长度；
// 偏移与长度配合正文读取请求定位该分类在资料文件中的正文区间。
type CompanyCategory struct {
	Name     string // 分类名称，来自 64 字节定长 `GBK` 字段解码。
	Filename string // 资料文件名，来自 80 字节定长 `GBK` 字段解码。
	Start    uint32 // 正文在资料文件内的起始偏移（字节，4 字节）。
	Length   uint32 // 正文字节长度（4 字节）。
}

// NewGetCompanyCategory 创建并初始化公司资料目录查询请求对象。
//
// 参数 `req` 携带 `Market` 市场代码与 `Code` 证券代码；传 `nil` 时仅初始化协议骨架。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetCompanyCategory(req *GetCompanyCategoryRequest) *GetCompanyCategory {
	obj := new(GetCompanyCategory)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetCompanyCategoryRequest)
	obj.reply = new(GetCompanyCategoryReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_COMPANYCATEGORY
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetCompanyCategory) applyRequest(req *GetCompanyCategoryRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetCompanyCategory) BuildRequest() ([]byte, error) {
	// 包头定长 14 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 14
	obj.reqHeader.PkgLen2 = 14

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的公司资料目录并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 0:2 为条目条数；其后每条记录固定 152 字节：0:64 分类名称，64:144 文件名，
// 144:148 起始偏移，148:152 内容长度。
func (obj *GetCompanyCategory) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 0:2 目录条目条数。
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])

	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条记录固定 152 字节，起始偏移为 2 字节报文头加序号乘 152。
		base := 2 + int(i)*152
		obj.reply.Categories = append(obj.reply.Categories, CompanyCategory{
			// 0:64 定长分类名，解码时截断尾部填充并去除空白。
			Name: decodeFixedGBK(data[base : base+64]),
			// 64:144 定长 80 字节文件名。
			Filename: decodeFixedGBK(data[base+64 : base+144]),
			// 144:148 正文起始偏移。
			Start: binary.LittleEndian.Uint32(data[base+144 : base+148]),
			// 148:152 正文长度。
			Length: binary.LittleEndian.Uint32(data[base+148 : base+152]),
		})
	}
	return nil
}

// Response 返回解析完成后的公司资料目录响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetCompanyCategory) Response() *GetCompanyCategoryReply {
	return obj.reply
}

// GetCompanyContent 封装按文件名与偏移读取公司 F10 资料正文内容的请求与响应处理流程。
//
// 配合目录查询返回的文件名、偏移与长度发起二次请求。
type GetCompanyContent struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetCompanyContentRequest
	reply      *GetCompanyContentReply
}

// GetCompanyContentRequest 是公司资料正文读取的请求参数。
type GetCompanyContentRequest struct {
	Market   uint16   // 市场代码（2 字节），取值与行情协议市场枚举一致。
	Code     [6]byte  // 6 字节定长证券代码，尾部以零字节填充。
	Zero     uint16   // 协议保留占位，发送时填 0，语义待确认。
	Filename [80]byte // 80 字节定长资料文件名，来自目录条目返回值。
	Start    uint32   // 正文在资料文件内的起始偏移（字节，4 字节）。
	Length   uint32   // 本次请求读取的正文字节长度（4 字节）。
	Zero2    uint32   // 协议保留占位，发送时填 0，语义待确认。
}

// GetCompanyContentReply 是公司资料正文读取的响应数据。
type GetCompanyContentReply struct {
	Market   uint16 // 回显市场代码（2 字节）。
	Code     string // 回显证券代码。
	MarketOR uint16 // 回显市场字段，语义待确认。
	Length   uint16 // 实际返回的正文字节长度（2 字节）。
	Content  string // 资料正文，`GBK` 解码并剔除尾部填充后的文本。
}

// NewGetCompanyContent 创建并初始化公司资料正文读取请求对象。
//
// 参数 `req` 携带市场、证券代码、目标文件名、正文偏移与读取长度；传 `nil` 时仅初始化协议骨架。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetCompanyContent(req *GetCompanyContentRequest) *GetCompanyContent {
	obj := new(GetCompanyContent)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetCompanyContentRequest)
	obj.reply = new(GetCompanyContentReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_COMPANYCONTENT
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetCompanyContent) applyRequest(req *GetCompanyContentRequest) {
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetCompanyContent) BuildRequest() ([]byte, error) {
	// 包头定长 104 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 104
	obj.reqHeader.PkgLen2 = 104

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的公司资料正文并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 12 字节定长头：0:2 市场代码，2:8 证券代码，8:10 回显市场字段（语义待确认），
// 10:12 正文字节长度；12 起为对应长度的正文字节，尾部填充需剔除。
func (obj *GetCompanyContent) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 0:2 市场代码。
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	// 2:8 六字节证券代码。
	obj.reply.Code = Utf8ToGbk(data[2:8])
	// 8:10 回显市场字段，语义待确认。
	obj.reply.MarketOR = binary.LittleEndian.Uint16(data[8:10])
	// 10:12 正文字节长度。
	obj.reply.Length = binary.LittleEndian.Uint16(data[10:12])
	// 12 起按长度截取正文，解码后剔除尾部填充。
	obj.reply.Content = strings.TrimRight(Utf8ToGbk(data[12:12+obj.reply.Length]), "\x00")
	return nil
}

// Response 返回解析完成后的公司资料正文响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetCompanyContent) Response() *GetCompanyContentReply {
	return obj.reply
}

// GetFinanceInfo 封装上市公司财务指标与股本概况查询的完整请求与响应处理流程。
type GetFinanceInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetFinanceInfoRequest
	reply      *GetFinanceInfoReply
}

// GetFinanceInfoRequest 是财务概况查询的请求参数。
type GetFinanceInfoRequest struct {
	One    uint16  // 固定协议标记；传 0 时由构造函数回填默认值 1。
	Market uint8   // 市场代码（1 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
}

// GetFinanceInfoReply 是财务概况查询的响应数据，含股本、资产负债、利润等财务字段。
//
// 除日期、编号类字段外，金额与股本字段均以协议原始 `float32` 顺序返回；
// 资产负债类字段的计量单位口径未公开，调用方不应自行换算。
type GetFinanceInfoReply struct {
	Num                 uint16  // 记录编号（2 字节），协议序号语义待确认。
	Market              uint8   // 回显市场代码（1 字节）。
	Code                string  // 回显证券代码。
	FloatShares         float32 // 流通股本（万股）。
	Province            uint16  // 所属省份编号，编号表语义待确认。
	Industry            uint16  // 所属行业编号，编号表语义待确认。
	UpdatedDate         uint32  // 财务数据更新日期，按 `YYYYMMDD` 整数编码。
	IPODate             uint32  // 上市日期，按 `YYYYMMDD` 整数编码。
	TotalShares         float32 // 总股本（万股）。
	StateShares         float32 // 国有股股本（万股），口径语义待确认。
	SponsorLegalShares  float32 // 发起人法人股股本（万股），口径语义待确认。
	LegalShares         float32 // 法人股股本（万股），口径语义待确认。
	BShares             float32 // B 股股本（万股）。
	HShares             float32 // H 股股本（万股）。
	EPS                 float32 // 每股收益（元/股）。
	TotalAssets         float32 // 资产总额，金额口径以协议返回为准。
	CurrentAssets       float32 // 流动资产合计，金额口径以协议返回为准。
	FixedAssets         float32 // 固定资产合计，金额口径以协议返回为准。
	IntangibleAssets    float32 // 无形资产，金额口径以协议返回为准。
	ShareholderCount    float32 // 股东户数，单位口径以协议返回为准。
	CurrentLiabilities  float32 // 流动负债合计，金额口径以协议返回为准。
	LongTermLiabilities float32 // 长期负债合计，金额口径以协议返回为准。
	CapitalReserve      float32 // 资本公积，金额口径以协议返回为准。
	TotalEquity         float32 // 股东权益合计，金额口径以协议返回为准。
	OperatingRevenue    float32 // 主营业务收入，金额口径以协议返回为准。
	OperatingCost       float32 // 主营业务成本，金额口径以协议返回为准。
	AccountsReceivable  float32 // 应收账款，金额口径以协议返回为准。
	OperatingProfit     float32 // 营业利润，金额口径以协议返回为准。
	InvestmentIncome    float32 // 投资收益，金额口径以协议返回为准。
	NetCashFlow         float32 // 现金流量净额，金额口径以协议返回为准。
	TotalCashInflow     float32 // 现金流入合计，金额口径以协议返回为准。
	Inventory           float32 // 存货，金额口径以协议返回为准。
	TotalProfit         float32 // 利润总额，金额口径以协议返回为准。
	AfterTaxProfit      float32 // 税后利润，金额口径以协议返回为准。
	NetProfit           float32 // 净利润，金额口径以协议返回为准。
	UndistributedProfit float32 // 未分配利润，金额口径以协议返回为准。
	NetAssetsPerShare   float32 // 每股净资产（元/股）。
	Reserved2           float32 // 协议保留字段，语义待确认。
}

// NewGetFinanceInfo 创建并初始化财务概况查询请求对象。
//
// 参数 `req` 携带 `Market` 市场代码与 `Code` 证券代码；`One` 传 0 时回填默认 1；
// 传 `nil` 时仅初始化协议骨架并写入默认标记。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetFinanceInfo(req *GetFinanceInfoRequest) *GetFinanceInfo {
	obj := new(GetFinanceInfo)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetFinanceInfoRequest)
	obj.reply = new(GetFinanceInfoReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_FINANCEINFO
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetFinanceInfo) applyRequest(req *GetFinanceInfoRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetFinanceInfo) BuildRequest() ([]byte, error) {
	// 包头定长 11 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 11
	obj.reqHeader.PkgLen2 = 11

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 按协议顺序解析服务端返回的财务字段并填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 报文顺序流式读取，读完一个字段后读指针自动前进；绑定字段的顺序必须与协议一致；
// 任一字段读取长度不足时返回读取错误。
func (obj *GetFinanceInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 读取器，读完字段自动推进读指针。
	reader := bytes.NewReader(data)

	// 头部：2 字节记录编号。
	if err := binary.Read(reader, binary.LittleEndian, &obj.reply.Num); err != nil {
		return err
	}
	// 1 字节市场代码。
	if err := binary.Read(reader, binary.LittleEndian, &obj.reply.Market); err != nil {
		return err
	}
	// 6 字节定长证券代码，尾部填充需剔除。
	var code [6]byte
	if err := binary.Read(reader, binary.LittleEndian, &code); err != nil {
		return err
	}
	obj.reply.Code = strings.TrimRight(string(code[:]), "\x00")

	// 后续财务字段按协议固定顺序依次绑定到响应结构体。
	fields := []interface{}{
		&obj.reply.FloatShares,
		&obj.reply.Province,
		&obj.reply.Industry,
		&obj.reply.UpdatedDate,
		&obj.reply.IPODate,
		&obj.reply.TotalShares,
		&obj.reply.StateShares,
		&obj.reply.SponsorLegalShares,
		&obj.reply.LegalShares,
		&obj.reply.BShares,
		&obj.reply.HShares,
		&obj.reply.EPS,
		&obj.reply.TotalAssets,
		&obj.reply.CurrentAssets,
		&obj.reply.FixedAssets,
		&obj.reply.IntangibleAssets,
		&obj.reply.ShareholderCount,
		&obj.reply.CurrentLiabilities,
		&obj.reply.LongTermLiabilities,
		&obj.reply.CapitalReserve,
		&obj.reply.TotalEquity,
		&obj.reply.OperatingRevenue,
		&obj.reply.OperatingCost,
		&obj.reply.AccountsReceivable,
		&obj.reply.OperatingProfit,
		&obj.reply.InvestmentIncome,
		&obj.reply.NetCashFlow,
		&obj.reply.TotalCashInflow,
		&obj.reply.Inventory,
		&obj.reply.TotalProfit,
		&obj.reply.AfterTaxProfit,
		&obj.reply.NetProfit,
		&obj.reply.UndistributedProfit,
		&obj.reply.NetAssetsPerShare,
		&obj.reply.Reserved2,
	}
	// 逐个读取财务字段；任一字段长度不足即返回读取错误。
	for _, field := range fields {
		if err := binary.Read(reader, binary.LittleEndian, field); err != nil {
			return err
		}
	}
	return nil
}

// Response 返回解析完成后的财务概况响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetFinanceInfo) Response() *GetFinanceInfoReply {
	return obj.reply
}

// GetXDXRInfo 封装除权除息、送配股、增发、股本变动等历史权益信息查询的请求与响应处理流程。
type GetXDXRInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetXDXRInfoRequest
	reply      *GetXDXRInfoReply
}

// GetXDXRInfoRequest 是除权除息信息查询的请求参数。
type GetXDXRInfoRequest struct {
	One    uint16  // 固定协议标记；传 0 时由构造函数回填默认值 1。
	Market uint8   // 市场代码（1 字节），取值与行情协议市场枚举一致。
	Code   [6]byte // 6 字节定长证券代码，尾部以零字节填充。
}

// GetXDXRInfoReply 是除权除息信息查询的响应数据。
type GetXDXRInfoReply struct {
	Market   uint8      // 回显市场代码（1 字节）。
	MarketOR uint16     // 回显市场字段，语义待确认。
	Code     string     // 回显证券代码。
	Count    uint16     // 服务端返回的权益记录条数。
	List     []XDXRItem // 解析后的权益变动记录列表，按日期倒序排列。
}

// XDXRItem 表示一条除权除息或股本变动记录，按类别填充不同的价格与股本字段。
// 未被当前类别使用的槽位保持空指针，不赋值。
type XDXRItem struct {
	Market          uint8     // 记录所属市场代码。
	Code            string    // 记录对应证券代码。
	Unknown         uint8     // 未确认字节，语义待确认。
	Date            time.Time // 权益变动日期。
	Category        uint8     // 变动类别编号，取值见 `xdxrCategoryName` 的映射表。
	Name            string    // 变动类别中文名称，由类别编号映射得到。
	Fenhong         *float32  // 每股分红（元/股）；不适用类别为空指针。
	Peigujia        *float32  // 配股价（元/股）；不适用类别为空指针。
	Songzhuangu     *float32  // 送转股比例，单位口径语义待确认；不适用类别为空指针。
	Peigu           *float32  // 配股比例，单位口径语义待确认；不适用类别为空指针。
	Suogu           *float32  // 缩股比例，单位口径语义待确认；不适用类别为空指针。
	Xingquanjia     *float32  // 权证行权价（元/股）；不适用类别为空指针。
	Fenshu          *float32  // 权证份数；不适用类别为空指针。
	PreFloatShares  *float32  // 变动前流通股本，单位口径以协议返回为准；不适用类别为空指针。
	PreTotalShares  *float32  // 变动前总股本，单位口径以协议返回为准；不适用类别为空指针。
	PostFloatShares *float32  // 变动后流通股本，单位口径以协议返回为准；不适用类别为空指针。
	PostTotalShares *float32  // 变动后总股本，单位口径以协议返回为准；不适用类别为空指针。
}

// NewGetXDXRInfo 创建并初始化除权除息信息查询请求对象。
//
// 参数 `req` 携带 `Market` 市场代码与 `Code` 证券代码；`One` 传 0 时回填默认 1；
// 传 `nil` 时仅初始化协议骨架并写入默认标记。
// 返回初始化完成、可直接调用 `BuildRequest` 的协议对象。
func NewGetXDXRInfo(req *GetXDXRInfoRequest) *GetXDXRInfo {
	obj := new(GetXDXRInfo)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetXDXRInfoRequest)
	obj.reply = new(GetXDXRInfoReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_XDXRINFO
	obj.request.One = 1
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *GetXDXRInfo) applyRequest(req *GetXDXRInfoRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

// BuildRequest 将请求头与请求参数编码为协议字节流。
//
// 先回填定长包头长度，再依次写入请求头与请求参数；成功返回可直接发送的字节流，
// 写出失败时返回错误。
func (obj *GetXDXRInfo) BuildRequest() ([]byte, error) {
	// 包头定长 11 字节，两个长度校验位保持一致。
	obj.reqHeader.PkgLen1 = 11
	obj.reqHeader.PkgLen2 = 11

	buf := new(bytes.Buffer)
	// 请求头与请求参数均拼装进同一缓冲。
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

// ParseResponse 解析服务端返回的除权除息记录并按日期倒序填充响应对象。
//
// 参数 `header` 为协议响应头，`data` 为去除响应头后的报文体。
// 公共头 11 字节：0 市场，1:3 回显市场，3:9 代码，9:11 记录条数。
// 每条记录固定 29 字节：市场、代码、保留字节、日期、类别编号，以及四个按类别解释的数值槽位。
func (obj *GetXDXRInfo) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header

	// 0 字节：市场代码。
	obj.reply.Market = data[0]
	// 1:3 回显市场字段，语义待确认。
	obj.reply.MarketOR = binary.LittleEndian.Uint16(data[1:3])
	// 3:9 六字节证券代码。
	obj.reply.Code = Utf8ToGbk(data[3:9])
	// 9:11 权益记录条数。
	obj.reply.Count = binary.LittleEndian.Uint16(data[9:11])

	for i := uint16(0); i < obj.reply.Count; i++ {
		// 每条记录固定 29 字节，起始偏移为 11 字节公共头加序号乘 29。
		pos := 11 + int(i)*29
		item := XDXRItem{
			Market:   data[pos],
			Code:     Utf8ToGbk(data[pos+1 : pos+7]),
			Unknown:  data[pos+7],
			Category: data[pos+12],
			// 类别编号映射为中文类别名称，未识别类别返回空串。
			Name: xdxrCategoryName(data[pos+12]),
		}
		// 日期编码字段，解析为年月日时间。
		item.Date, _ = parseYMD(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		// 记录尾部四个连续数值槽位，具体含义由类别决定。
		left := data[pos+13 : pos+29]
		a := float32FromBytes(left[0:4])
		b := float32FromBytes(left[4:8])
		c := float32FromBytes(left[8:12])
		d := float32FromBytes(left[12:16])
		// 按类别把槽位映射到对应字段；未列出的类别走默认股本变化分支。
		switch item.Category {
		case 1:
			item.Fenhong = &a
			item.Peigujia = &b
			item.Songzhuangu = &c
			item.Peigu = &d
		case 11, 12:
			item.Suogu = &c
		case 13, 14:
			item.Xingquanjia = &a
			item.Fenshu = &c
		default:
			item.PreFloatShares = &a
			item.PreTotalShares = &b
			item.PostFloatShares = &c
			item.PostTotalShares = &d
		}
		obj.reply.List = append(obj.reply.List, item)
	}
	// 按权益日期倒序排列（最新一次除权除息排在最前）。
	if len(obj.reply.List) > 0 {
		sort.Slice(obj.reply.List, func(i, j int) bool {
			return obj.reply.List[i].Date.After(obj.reply.List[j].Date)
		})
	}
	return nil
}

// Response 返回解析完成后的除权除息响应数据。
// 未调用 `ParseResponse` 前，返回对象的各字段保持零值。
func (obj *GetXDXRInfo) Response() *GetXDXRInfoReply {
	return obj.reply
}

// decodeFixedGBK 解析定长 `GBK` 字符串：先按首个零字节截断填充区，再解码并去除首尾空白。
//
// 参数 `data` 为定长原始字节；返回解码后的纯文本；入参为空时返回空串。
func decodeFixedGBK(data []byte) string {
	if idx := bytes.IndexByte(data, 0x00); idx >= 0 {
		data = data[:idx]
	}
	return strings.TrimSpace(Utf8ToGbk(data))
}

// float32FromBytes 将 4 字节位模式还原为浮点数。
//
// 参数 `data` 必须至少包含 4 字节；返回按 `float32` 位模式解释后的浮点值。
func float32FromBytes(data []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}

// xdxrCategoryName 将协议权益类别编号映射为中文名称。
//
// 参数 `category` 为报文中的类别编号；命中已知编号时返回对应中文名称，
// 未识别编号返回空串。
func xdxrCategoryName(category uint8) string {
	switch category {
	case 1:
		return "除权除息"
	case 2:
		return "送配股上市"
	case 3:
		return "非流通股上市"
	case 4:
		return "未知股本变动"
	case 5:
		return "股本变化"
	case 6:
		return "增发新股"
	case 7:
		return "股份回购"
	case 8:
		return "增发新股上市"
	case 9:
		return "转配股上市"
	case 10:
		return "可转债上市"
	case 11:
		return "扩缩股"
	case 12:
		return "非流通股缩股"
	case 13:
		return "送认购权证"
	case 14:
		return "送认沽权证"
	default:
		return ""
	}
}

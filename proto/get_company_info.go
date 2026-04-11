package proto

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"time"
)

type GetCompanyCategory struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetCompanyCategoryRequest
	reply      *GetCompanyCategoryReply
}

type GetCompanyCategoryRequest struct {
	Market uint16
	Code   [6]byte
	Zero   uint32
}

type GetCompanyCategoryReply struct {
	Count      uint16
	Categories []CompanyCategory
}

type CompanyCategory struct {
	Name     string
	Filename string
	Start    uint32
	Length   uint32
}

func NewGetCompanyCategory() *GetCompanyCategory {
	obj := new(GetCompanyCategory)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetCompanyCategoryRequest)
	obj.reply = new(GetCompanyCategoryReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_COMPANYCATEGORY
	return obj
}

func (obj *GetCompanyCategory) SetParams(req *GetCompanyCategoryRequest) {
	obj.request = req
}

func (obj *GetCompanyCategory) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 14
	obj.reqHeader.PkgLen2 = 14

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetCompanyCategory) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	obj.reply.Count = binary.LittleEndian.Uint16(data[:2])

	for i := uint16(0); i < obj.reply.Count; i++ {
		base := 2 + int(i)*152
		obj.reply.Categories = append(obj.reply.Categories, CompanyCategory{
			Name:     decodeFixedGBK(data[base : base+64]),
			Filename: decodeFixedGBK(data[base+64 : base+144]),
			Start:    binary.LittleEndian.Uint32(data[base+144 : base+148]),
			Length:   binary.LittleEndian.Uint32(data[base+148 : base+152]),
		})
	}
	return nil
}

func (obj *GetCompanyCategory) Reply() *GetCompanyCategoryReply {
	return obj.reply
}

type GetCompanyContent struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetCompanyContentRequest
	reply      *GetCompanyContentReply
}

type GetCompanyContentRequest struct {
	Market   uint16
	Code     [6]byte
	Zero     uint16
	Filename [80]byte
	Start    uint32
	Length   uint32
	Zero2    uint32
}

type GetCompanyContentReply struct {
	Market   uint16
	Code     string
	MarketOR uint16
	Length   uint16
	Content  string
}

func NewGetCompanyContent() *GetCompanyContent {
	obj := new(GetCompanyContent)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetCompanyContentRequest)
	obj.reply = new(GetCompanyContentReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_COMPANYCONTENT
	return obj
}

func (obj *GetCompanyContent) SetParams(req *GetCompanyContentRequest) {
	obj.request = req
}

func (obj *GetCompanyContent) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 104
	obj.reqHeader.PkgLen2 = 104

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetCompanyContent) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	obj.reply.Market = binary.LittleEndian.Uint16(data[:2])
	obj.reply.Code = Utf8ToGbk(data[2:8])
	obj.reply.MarketOR = binary.LittleEndian.Uint16(data[8:10])
	obj.reply.Length = binary.LittleEndian.Uint16(data[10:12])
	obj.reply.Content = strings.TrimRight(Utf8ToGbk(data[12:12+obj.reply.Length]), "\x00")
	return nil
}

func (obj *GetCompanyContent) Reply() *GetCompanyContentReply {
	return obj.reply
}

type GetFinanceInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetFinanceInfoRequest
	reply      *GetFinanceInfoReply
}

type GetFinanceInfoRequest struct {
	One    uint16
	Market uint8
	Code   [6]byte
}

type GetFinanceInfoReply struct {
	Num                 uint16
	Market              uint8
	Code                string
	FloatShares         float32
	Province            uint16
	Industry            uint16
	UpdatedDate         uint32
	IPODate             uint32
	TotalShares         float32
	StateShares         float32
	SponsorLegalShares  float32
	LegalShares         float32
	BShares             float32
	HShares             float32
	EPS                 float32
	TotalAssets         float32
	CurrentAssets       float32
	FixedAssets         float32
	IntangibleAssets    float32
	ShareholderCount    float32
	CurrentLiabilities  float32
	LongTermLiabilities float32
	CapitalReserve      float32
	TotalEquity         float32
	OperatingRevenue    float32
	OperatingCost       float32
	AccountsReceivable  float32
	OperatingProfit     float32
	InvestmentIncome    float32
	NetCashFlow         float32
	TotalCashInflow     float32
	Inventory           float32
	TotalProfit         float32
	AfterTaxProfit      float32
	NetProfit           float32
	UndistributedProfit float32
	NetAssetsPerShare   float32
	Reserved2           float32
}

func NewGetFinanceInfo() *GetFinanceInfo {
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
	return obj
}

func (obj *GetFinanceInfo) SetParams(req *GetFinanceInfoRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *GetFinanceInfo) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 11
	obj.reqHeader.PkgLen2 = 11

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetFinanceInfo) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	reader := bytes.NewReader(data)

	if err := binary.Read(reader, binary.LittleEndian, &obj.reply.Num); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &obj.reply.Market); err != nil {
		return err
	}
	var code [6]byte
	if err := binary.Read(reader, binary.LittleEndian, &code); err != nil {
		return err
	}
	obj.reply.Code = strings.TrimRight(string(code[:]), "\x00")

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
	for _, field := range fields {
		if err := binary.Read(reader, binary.LittleEndian, field); err != nil {
			return err
		}
	}
	return nil
}

func (obj *GetFinanceInfo) Reply() *GetFinanceInfoReply {
	return obj.reply
}

type GetXDXRInfo struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetXDXRInfoRequest
	reply      *GetXDXRInfoReply
}

type GetXDXRInfoRequest struct {
	One    uint16
	Market uint8
	Code   [6]byte
}

type GetXDXRInfoReply struct {
	Market   uint8
	MarketOR uint16
	Code     string
	Count    uint16
	List     []XDXRItem
}

type XDXRItem struct {
	Market          uint8
	Code            string
	Unknown         uint8
	Date            time.Time
	Category        uint8
	Name            string
	Fenhong         *float32
	Peigujia        *float32
	Songzhuangu     *float32
	Peigu           *float32
	Suogu           *float32
	Xingquanjia     *float32
	Fenshu          *float32
	PreFloatShares  *float32
	PreTotalShares  *float32
	PostFloatShares *float32
	PostTotalShares *float32
}

func NewGetXDXRInfo() *GetXDXRInfo {
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
	return obj
}

func (obj *GetXDXRInfo) SetParams(req *GetXDXRInfoRequest) {
	if req.One == 0 {
		req.One = 1
	}
	obj.request = req
}

func (obj *GetXDXRInfo) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 11
	obj.reqHeader.PkgLen2 = 11

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetXDXRInfo) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)

	obj.reply.Market = data[0]
	obj.reply.MarketOR = binary.LittleEndian.Uint16(data[1:3])
	obj.reply.Code = Utf8ToGbk(data[3:9])
	obj.reply.Count = binary.LittleEndian.Uint16(data[9:11])

	for i := uint16(0); i < obj.reply.Count; i++ {
		pos := 11 + int(i)*29
		item := XDXRItem{
			Market:   data[pos],
			Code:     Utf8ToGbk(data[pos+1 : pos+7]),
			Unknown:  data[pos+7],
			Category: data[pos+12],
			Name:     xdxrCategoryName(data[pos+12]),
		}
		item.Date, _ = parseYMD(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		left := data[pos+13 : pos+29]
		a := float32FromBytes(left[0:4])
		b := float32FromBytes(left[4:8])
		c := float32FromBytes(left[8:12])
		d := float32FromBytes(left[12:16])
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
	return nil
}

func (obj *GetXDXRInfo) Reply() *GetXDXRInfoReply {
	return obj.reply
}

func decodeFixedGBK(data []byte) string {
	if idx := bytes.IndexByte(data, 0x00); idx >= 0 {
		data = data[:idx]
	}
	return strings.TrimSpace(Utf8ToGbk(data))
}

func float32FromBytes(data []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}

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

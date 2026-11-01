package gotdx

// MACFieldSelector 是参与构造 MAC 动态字段位图的选择项接口。
//
// 单个字段位（`MACFieldBit`）与一组字段预设（`MACPresetField`）均实现该接口；调用方将
// 若干选择项传给 `MACFieldBitmap`，由其组合出最终位图。位图中被置位的字段才会在服务端
// 响应中返回。
type MACFieldSelector interface {
	// applyMACFieldBitmap 把本选择项覆盖的字段位写入目标位图。
	//
	// 参数 `bitmap` 指向待写入的 20 字节位图；实现仅置位，不清除位图中已有的其他位。
	applyMACFieldBitmap(*[20]byte)
}

// MACFieldBit 表示 MAC 动态字段位图中的单个字段位，底层为协议位号，有效范围为 0–159
// （对应 20 字节位图的 160 个二进制位）。
//
// 位号与 `proto/mac_board_members_dynamic.go` 中的字段映射表一一对应；超出位图容量的位号
// 在置位时会被忽略，不会产生越界写入。
type MACFieldBit uint8

// MACField* 位图字段：MAC 动态字段位图中可请求的行情字段，位号与
// `proto/mac_board_members_dynamic.go` 的字段映射表一一对应；位图中置位后，服务端才会返回
// 对应字段。每个字段的具体含义、单位与特殊取值见各常量行尾注释。
//
// 标注“语义待确认”的字段协议含义未在映射表中确证，不应据此推断协议细节；块末若干常量为
// 历史遗留别名，指向同一字段位，保留以兼容旧调用，对应关系见各别名常量行尾注释。
const (
	MACFieldPreClose            MACFieldBit = 0x0                       // 昨收盘价，上一交易日收盘价格
	MACFieldOpen                MACFieldBit = 0x1                       // 今日开盘价
	MACFieldHigh                MACFieldBit = 0x2                       // 今日最高价
	MACFieldLow                 MACFieldBit = 0x3                       // 今日最低价
	MACFieldClose               MACFieldBit = 0x4                       // 今日收盘价
	MACFieldVol                 MACFieldBit = 0x5                       // 今日成交量，单位股
	MACFieldVolRatio            MACFieldBit = 0x6                       // 量比，当前成交活跃度与近期均量之比
	MACFieldAmount              MACFieldBit = 0x7                       // 今日成交总金额，单位元
	MACFieldInsideVolume        MACFieldBit = 0x8                       // 内盘成交量，主动卖出方向成交的股数
	MACFieldOutsideVolume       MACFieldBit = 0x9                       // 外盘成交量，主动买入方向成交的股数
	MACFieldTotalShares         MACFieldBit = 0xa                       // 总股本，单位万股
	MACFieldFloatShares         MACFieldBit = 0xb                       // 流通股本，单位万股；港股场景别名 total_shares_hk
	MACFieldEPS                 MACFieldBit = 0xc                       // 每股收益，净利润除以总股本
	MACFieldNetAssets           MACFieldBit = 0xd                       // 每股净资产
	MACFieldSecurityTypePrice   MACFieldBit = 0xe                       // 证券类型价（别名 action_price，语义待确认）
	MACFieldTotalMarketCapAb    MACFieldBit = 0xf                       // A/B 股总市值
	MACFieldPEDynamic           MACFieldBit = 0x10                      // 动态市盈率
	MACFieldBidPrice            MACFieldBit = 0x11                      // 买一报价
	MACFieldAskPrice            MACFieldBit = 0x12                      // 卖一报价
	MACFieldServerUpdateDate    MACFieldBit = 0x13                      // 行情服务器更新日期，格式 YYYYMMDD
	MACFieldServerUpdateTime    MACFieldBit = 0x14                      // 行情服务器更新时间，格式 HHMMSS
	MACFieldLotSizeInfo         MACFieldBit = 0x15                      // 每手相关信息（映射表标注未确定，语义待确认）
	MACFieldBoardStrength       MACFieldBit = 0x16                      // 板块强度，板块内上涨与下跌家数之差
	MACFieldDividendYield       MACFieldBit = 0x17                      // 每股股息，单位元
	MACFieldBidVolume           MACFieldBit = 0x18                      // 买一挂单量
	MACFieldAskVolume           MACFieldBit = 0x19                      // 卖一挂单量
	MACFieldLastVolume          MACFieldBit = 0x1a                      // 最新一笔成交量，即现量
	MACFieldTurnover            MACFieldBit = 0x1b                      // 换手率
	MACFieldIndustry            MACFieldBit = 0x1c                      // 行业分类代码
	MACFieldIndustryChangeUp    MACFieldBit = 0x1d                      // 所属行业涨跌幅
	MACFieldStockTagFlags       MACFieldBit = 0x1e                      // 股票标签位图，多个标签按位组合
	MACFieldDecimalPoint        MACFieldBit = 0x1f                      // 价格数据精度，记录小数位标志
	MACFieldBuyPriceLimit       MACFieldBit = 0x20                      // 涨停价
	MACFieldSellPriceLimit      MACFieldBit = 0x21                      // 跌停价
	MACFieldPriceDecimalInfo    MACFieldBit = 0x22                      // 价格精度标志（语义待确认）
	MACFieldLotSize             MACFieldBit = 0x23                      // 所属地区板块（A 股）/ 每手股数（港股），含义随品种双义
	MACFieldPreIOPV             MACFieldBit = 0x24                      // 昨日基金份额参考净值，即昨 IOPV
	MACFieldSpeedPct            MACFieldBit = 0x25                      // 涨速，单位时间内价格变动幅度
	MACFieldAvgPrice            MACFieldBit = 0x26                      // 成交均价，成交额除以成交量
	MACFieldIOPV                MACFieldBit = 0x27                      // 基金份额参考净值（IOPV）
	MACFieldPETTMVolRelated     MACFieldBit = 0x28                      // 前参考价，美股品种适用
	MACFieldExPricePlaceholder  MACFieldBit = 0x29                      // 前金额参考占位（别名 close_placeholder）
	MACFieldOperatingRevenue    MACFieldBit = 0x2a                      // 营业收入，单位万元
	MACFieldFlagKCB             MACFieldBit = 0x2b                      // 科创板标志位
	MACFieldFlagBJ              MACFieldBit = 0x2c                      // 北交所标志位
	MACFieldCirculatingCapitalZ MACFieldBit = 0x2d                      // 流通股本 Z，单位万股（语义待确认）
	MACFieldAfterHoursVolume    MACFieldBit = 0x2e                      // 盘后成交量
	MACFieldPETTM               MACFieldBit = 0x30                      // 滚动市盈率（PE TTM）
	MACFieldPEStatic            MACFieldBit = 0x31                      // 静态市盈率
	MACFieldIndexMetric         MACFieldBit = 0x37                      // 指数指标值
	MACFieldMainNetAmount       MACFieldBit = 0x38                      // 今日主力净流入金额
	MACFieldBidAskRatio         MACFieldBit = 0x39                      // 委比，买卖盘挂单力量对比
	MACFieldNonIndexFlag        MACFieldBit = 0x3a                      // 非指数标志位
	MACFieldChange20DPct        MACFieldBit = 0x3b                      // 近 20 日涨幅百分比
	MACFieldYTDPct              MACFieldBit = 0x3c                      // 年初至今涨幅百分比
	MACFieldStockClassCode      MACFieldBit = 0x3e                      // 证券子分类代码
	MACFieldPercentBase         MACFieldBit = 0x3f                      // 百分比基底（语义待确认）
	MACFieldMTDPct              MACFieldBit = 0x40                      // 月初至今涨幅百分比
	MACFieldChange1yPct         MACFieldBit = 0x41                      // 近一年涨幅百分比
	MACFieldPrevChangePct       MACFieldBit = 0x42                      // 昨日涨幅百分比
	MACFieldChange3DPct         MACFieldBit = 0x43                      // 近 3 日涨幅百分比
	MACFieldChange60DPct        MACFieldBit = 0x44                      // 近 60 日涨幅百分比
	MACFieldChange5DPct         MACFieldBit = 0x45                      // 近 5 日涨幅百分比
	MACFieldChange10DPct        MACFieldBit = 0x46                      // 近 10 日涨幅百分比
	MACFieldPrev2ChangePct      MACFieldBit = 0x47                      // 前日涨幅百分比
	MACFieldBid2Price           MACFieldBit = 0x48                      // 买二报价
	MACFieldAsk2Price           MACFieldBit = 0x49                      // 卖二报价
	MACFieldAHCode              MACFieldBit = 0x4a                      // 对应 A/H 股代码，位数不足前面补 0
	MACFieldUnknownCode         MACFieldBit = 0x4b                      // 未知代码，少部分为 6 位数字（语义待确认）
	MACFieldOpenAmount          MACFieldBit = 0x57                      // 开盘金额，单位元
	MACFieldAnnualLimitUpDays   MACFieldBit = 0x58                      // 年内涨停天数
	MACFieldActivity            MACFieldBit = 0x59                      // 活跃度
	MACFieldDividendYieldRate   MACFieldBit = 0x5b                      // 股息率百分比
	MACFieldConsecutiveUpDays   MACFieldBit = 0x5c                      // 连续上涨天数
	MACFieldLimitUpCount        MACFieldBit = 0x5d                      // 板块为涨停家数；个股复用为买二量（别名 Bid2Volume）
	MACFieldLimitDownCount      MACFieldBit = 0x5e                      // 板块为跌停家数；个股复用为卖二量（别名 Ask2Volume）
	MACFieldIndustrySub         MACFieldBit = 0x5f                      // 行业二级分类代码
	MACFieldAuctionBuyLimit     MACFieldBit = 0x66                      // 连续竞价买入上限价
	MACFieldAuctionSellLimit    MACFieldBit = 0x67                      // 连续竞价卖出下限价
	MACFieldVolSpeedPct         MACFieldBit = 0x68                      // 量涨速百分比
	MACFieldShortTurnoverPct    MACFieldBit = 0x69                      // 短周期换手率百分比
	MACFieldAmount2m            MACFieldBit = 0x6a                      // 近 2 分钟成交金额，单位元
	MACFieldMainNetAmountCopy   MACFieldBit = 0x6b                      // 今日主力净流入金额副本
	MACFieldMainNetRatio        MACFieldBit = 0x6c                      // 主力净比百分比
	MACFieldRetailNetAmount     MACFieldBit = 0x6d                      // 散户单增比（映射表原文，语义待确认）
	MACFieldMainNet5mAmount     MACFieldBit = 0x6e                      // 近 5 分钟主力净额
	MACFieldMainNet3DAmount     MACFieldBit = 0x6f                      // 近三日主力净额
	MACFieldMainNet5DAmount     MACFieldBit = 0x70                      // 近五日主力净额
	MACFieldMainNet10DAmount    MACFieldBit = 0x71                      // 近十日主买金额（映射表标注待确定）
	MACFieldMainBuyNetAmount    MACFieldBit = 0x72                      // 今日主买净额
	MACFieldDDX                 MACFieldBit = 0x73                      // 大单动向指标 DDX，具体计算口径语义待确认
	MACFieldDDY                 MACFieldBit = 0x74                      // 涨跌动因指标 DDY，具体计算口径语义待确认
	MACFieldDDZ                 MACFieldBit = 0x75                      // 大单差分指标 DDZ，具体计算口径语义待确认
	MACFieldDDF                 MACFieldBit = 0x76                      // DDF 资金指标，具体含义语义待确认
	MACFieldStockFlagA          MACFieldBit = 0x77                      // 个股标志位 A
	MACFieldStockFlagB          MACFieldBit = 0x78                      // 个股标志位 B，为 A 的副本
	MACFieldAuctionVolRatio     MACFieldBit = 0x7a                      // 竞价昨比（映射表原文，语义待确认）
	MACFieldPrevAmount          MACFieldBit = 0x7b                      // 昨日成交金额，单位元
	MACFieldRecentIndicator     MACFieldBit = 0x7d                      // 近日指标提示（语义待确认）
	MACFieldBid3Price           MACFieldBit = 0x80                      // 买三报价
	MACFieldBid4Price           MACFieldBit = 0x81                      // 买四报价
	MACFieldBid5Price           MACFieldBit = 0x82                      // 买五报价
	MACFieldAsk3Price           MACFieldBit = 0x83                      // 卖三报价
	MACFieldAsk4Price           MACFieldBit = 0x84                      // 卖四报价
	MACFieldAsk5Price           MACFieldBit = 0x85                      // 卖五报价
	MACFieldBid3Volume          MACFieldBit = 0x86                      // 买三挂单量
	MACFieldBid4Volume          MACFieldBit = 0x87                      // 买四挂单量
	MACFieldUpCount             MACFieldBit = 0x88                      // 板块为上涨家数；个股复用为买五量（别名 Bid5Volume）
	MACFieldAsk3Volume          MACFieldBit = 0x89                      // 卖三挂单量
	MACFieldAsk4Volume          MACFieldBit = 0x8a                      // 卖四挂单量
	MACFieldDownCount           MACFieldBit = 0x8b                      // 板块为下跌家数；个股复用为卖五量（别名 Ask5Volume）
	MACFieldBidAskDiff          MACFieldBit = 0x8c                      // 委差，买盘与卖盘挂单量之差
	MACFieldChangeUpType        MACFieldBit = 0x8d                      // 封板状态
	MACFieldSafetyScore         MACFieldBit = 0x8e                      // 安全分（别名 ConstantNegOne）
	MACFieldHighlightCount      MACFieldBit = 0x8f                      // 亮点数（别名 StockRating，个股评级）
	MACFieldChangeAt1000        MACFieldBit = 0x90                      // 10:00 时点的日内涨幅百分比
	MACFieldChangeAt1030        MACFieldBit = 0x91                      // 10:30 时点的日内涨幅百分比
	MACFieldChangeAt1100        MACFieldBit = 0x92                      // 11:00 时点的日内涨幅百分比
	MACFieldChangeAt1130        MACFieldBit = 0x93                      // 11:30 时点的日内涨幅百分比
	MACFieldChangeAt1330        MACFieldBit = 0x94                      // 13:30 时点的日内涨幅百分比
	MACFieldChangeAt1400        MACFieldBit = 0x95                      // 14:00 时点的日内涨幅百分比
	MACFieldChangeAt1430        MACFieldBit = 0x96                      // 14:30 时点的日内涨幅百分比
	MACFieldSomeBitmap          MACFieldBit = MACFieldStockTagFlags     // 历史遗留别名，等同 StockTagFlags
	MACFieldPreIPOV             MACFieldBit = MACFieldPreIOPV           // 历史遗留别名，等同 PreIOPV
	MACFieldIPOV                MACFieldBit = MACFieldIOPV              // 历史遗留别名，等同 IOPV
	MACFieldGEMStarInfo         MACFieldBit = MACFieldAfterHoursVolume  // 历史遗留别名，等同 AfterHoursVolume
	MACFieldDividendYieldPct    MACFieldBit = MACFieldDividendYieldRate // 历史遗留别名，等同 DividendYieldRate
	MACFieldConstantNegOne      MACFieldBit = MACFieldSafetyScore       // 历史遗留别名，等同 SafetyScore
	MACFieldStockRating         MACFieldBit = MACFieldHighlightCount    // 历史遗留别名，等同 HighlightCount
	MACFieldBid2Volume          MACFieldBit = MACFieldLimitUpCount      // 历史遗留别名，等同 LimitUpCount
	MACFieldAsk2Volume          MACFieldBit = MACFieldLimitDownCount    // 历史遗留别名，等同 LimitDownCount
	MACFieldBid5Volume          MACFieldBit = MACFieldUpCount           // 历史遗留别名，等同 UpCount
	MACFieldAsk5Volume          MACFieldBit = MACFieldDownCount         // 历史遗留别名，等同 DownCount
)

// applyMACFieldBitmap 将本字段位置入目标位图。
//
// 参数 `bitmap` 指向待写入的 20 字节位图；方法仅置位本字段对应的二进制位，不改动其他位。
// 位号超出 0–159 位图容量时静默忽略，避免构造预设位集时传入越位常量造成越界写入；方法不返回错误。
func (bit MACFieldBit) applyMACFieldBitmap(bitmap *[20]byte) {
	// 位图固定 20 字节共 160 位，越界位号直接丢弃，保证后续按位运算不会访问越界字节。
	if int(bit) >= len(bitmap)*8 {
		return
	}
	// 按 `bit/8` 定位所在字节、`bit%8` 定位字节内的位，先取移位掩码再按位或完成置位。
	bitmap[int(bit)/8] |= 1 << uint(int(bit)%8)
}

// MACPresetField 表示一组常用 MAC 动态字段预设：一个预设编号对应一组固定字段位，调用方
// 无需逐位列举字段，即可通过 `MACFieldBitmap` 选中整组字段。
type MACPresetField uint8

// MACPreset* 常用字段预设：一次性选中一组 MAC 动态字段，便于直接构造字段位图。各预设包含的
// 具体字段组见各常量行尾注释。
const (
	MACPresetNone        MACPresetField = 0  // 不选择任何字段，位图保持全零
	MACPresetOHLC        MACPresetField = 1  // 仅选中开盘、最高、最低、收盘四个价格字段
	MACPresetBasic       MACPresetField = 2  // 基础行情：开高低收、昨收、成交量
	MACPresetQuote       MACPresetField = 3  // 盘口报价：买一价、卖一价、买一量、卖一量、现量
	MACPresetVolume      MACPresetField = 4  // 量额换手：成交量、成交额、换手率、量比
	MACPresetFundamental MACPresetField = 5  // 基本面：总股本、流通股本、每股收益、净资产
	MACPresetEnhanced    MACPresetField = 6  // 增强行情：开高低收、成交量、流通股本、活跃度
	MACPresetAHCode      MACPresetField = 7  // A/H 辅助信息：开高低收、成交量、A/H 代码、每手、行业
	MACPresetBoardStats  MACPresetField = 8  // 板块涨跌统计：涨停、跌停、上涨、下跌家数
	MACPresetHandicap    MACPresetField = 9  // 五档盘口：五档买卖报价与对应挂单量
	MACPresetCommon      MACPresetField = 10 // 常用字段集合，覆盖普通行情查询所需字段
	MACPresetDebug       MACPresetField = 11 // 调试用，将位图 160 位全部置 1
	MACPresetAll         MACPresetField = 12 // 选中全部已定义字段位
)

// applyMACFieldBitmap 按预设编号把内置字段位集展开写入位图。
//
// 参数 `bitmap` 指向待写入的 20 字节位图；`None` 不写任何位，`Debug` 将整份位图全部置 1，
// 其余预设遍历各自字段位列表逐个置位；未识别的预设编号不写入任何位。方法仅置位不清除已有位，
// 不返回错误。
func (preset MACPresetField) applyMACFieldBitmap(bitmap *[20]byte) {
	switch preset {
	case MACPresetNone:
		return
	case MACPresetOHLC:
		for _, bit := range []MACFieldBit{MACFieldOpen, MACFieldHigh, MACFieldLow, MACFieldClose} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetBasic:
		for _, bit := range []MACFieldBit{MACFieldOpen, MACFieldHigh, MACFieldLow, MACFieldClose, MACFieldPreClose, MACFieldVol} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetQuote:
		for _, bit := range []MACFieldBit{MACFieldBidPrice, MACFieldAskPrice, MACFieldBidVolume, MACFieldAskVolume, MACFieldLastVolume} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetVolume:
		for _, bit := range []MACFieldBit{MACFieldVol, MACFieldAmount, MACFieldTurnover, MACFieldVolRatio} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetFundamental:
		for _, bit := range []MACFieldBit{MACFieldTotalShares, MACFieldFloatShares, MACFieldEPS, MACFieldNetAssets} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetEnhanced:
		for _, bit := range []MACFieldBit{MACFieldOpen, MACFieldHigh, MACFieldLow, MACFieldClose, MACFieldVol, MACFieldFloatShares, MACFieldActivity} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetAHCode:
		for _, bit := range []MACFieldBit{MACFieldOpen, MACFieldHigh, MACFieldLow, MACFieldClose, MACFieldVol, MACFieldAHCode, MACFieldLotSize, MACFieldIndustry} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetBoardStats:
		for _, bit := range []MACFieldBit{MACFieldLimitUpCount, MACFieldLimitDownCount, MACFieldUpCount, MACFieldDownCount} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetHandicap:
		for _, bit := range []MACFieldBit{MACFieldBidPrice, MACFieldBid2Price, MACFieldBid3Price, MACFieldBid4Price, MACFieldBid5Price, MACFieldAskPrice, MACFieldAsk2Price, MACFieldAsk3Price, MACFieldAsk4Price, MACFieldAsk5Price, MACFieldBidVolume, MACFieldLimitUpCount, MACFieldBid3Volume, MACFieldBid4Volume, MACFieldUpCount, MACFieldAskVolume, MACFieldLimitDownCount, MACFieldAsk3Volume, MACFieldAsk4Volume, MACFieldDownCount} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetCommon:
		for _, bit := range []MACFieldBit{MACFieldPreClose, MACFieldOpen, MACFieldHigh, MACFieldLow, MACFieldClose, MACFieldVol, MACFieldVolRatio, MACFieldAmount, MACFieldTotalShares, MACFieldFloatShares, MACFieldEPS, MACFieldNetAssets, MACFieldSecurityTypePrice, MACFieldTotalMarketCapAb, MACFieldPEDynamic, MACFieldLotSizeInfo, MACFieldDividendYield, MACFieldLastVolume, MACFieldTurnover, MACFieldStockTagFlags, MACFieldDecimalPoint, MACFieldBuyPriceLimit, MACFieldSellPriceLimit, MACFieldPriceDecimalInfo, MACFieldLotSize, MACFieldPreIOPV, MACFieldSpeedPct, MACFieldFlagKCB, MACFieldPETTM, MACFieldPEStatic, MACFieldMainNetAmount, MACFieldVolSpeedPct, MACFieldShortTurnoverPct, MACFieldCirculatingCapitalZ} {
			bit.applyMACFieldBitmap(bitmap)
		}
	case MACPresetDebug:
		// 调试场景需要请求全部字段，故将 20 字节全部填为 `0xff`，等价于 160 位全部置 1。
		for i := range bitmap {
			bitmap[i] = 0xff
		}
	case MACPresetAll:
		for _, bit := range []MACFieldBit{MACFieldPreClose, MACFieldOpen, MACFieldHigh, MACFieldLow, MACFieldClose, MACFieldVol, MACFieldVolRatio, MACFieldAmount, MACFieldInsideVolume, MACFieldOutsideVolume, MACFieldTotalShares, MACFieldFloatShares, MACFieldEPS, MACFieldNetAssets, MACFieldSecurityTypePrice, MACFieldTotalMarketCapAb, MACFieldPEDynamic, MACFieldBidPrice, MACFieldAskPrice, MACFieldServerUpdateDate, MACFieldServerUpdateTime, MACFieldLotSizeInfo, MACFieldBoardStrength, MACFieldDividendYield, MACFieldBidVolume, MACFieldAskVolume, MACFieldLastVolume, MACFieldTurnover, MACFieldIndustry, MACFieldIndustryChangeUp, MACFieldStockTagFlags, MACFieldDecimalPoint, MACFieldBuyPriceLimit, MACFieldSellPriceLimit, MACFieldPriceDecimalInfo, MACFieldLotSize, MACFieldPreIOPV, MACFieldSpeedPct, MACFieldAvgPrice, MACFieldIOPV, MACFieldPETTMVolRelated, MACFieldExPricePlaceholder, MACFieldOperatingRevenue, MACFieldFlagKCB, MACFieldFlagBJ, MACFieldCirculatingCapitalZ, MACFieldAfterHoursVolume, MACFieldPETTM, MACFieldPEStatic, MACFieldIndexMetric, MACFieldMainNetAmount, MACFieldBidAskRatio, MACFieldNonIndexFlag, MACFieldChange20DPct, MACFieldYTDPct, MACFieldStockClassCode, MACFieldPercentBase, MACFieldMTDPct, MACFieldChange1yPct, MACFieldPrevChangePct, MACFieldChange3DPct, MACFieldChange60DPct, MACFieldChange5DPct, MACFieldChange10DPct, MACFieldPrev2ChangePct, MACFieldBid2Price, MACFieldAsk2Price, MACFieldAHCode, MACFieldUnknownCode, MACFieldOpenAmount, MACFieldAnnualLimitUpDays, MACFieldActivity, MACFieldDividendYieldRate, MACFieldConsecutiveUpDays, MACFieldLimitUpCount, MACFieldLimitDownCount, MACFieldIndustrySub, MACFieldAuctionBuyLimit, MACFieldAuctionSellLimit, MACFieldVolSpeedPct, MACFieldShortTurnoverPct, MACFieldAmount2m, MACFieldMainNetAmountCopy, MACFieldMainNetRatio, MACFieldRetailNetAmount, MACFieldMainNet5mAmount, MACFieldMainNet3DAmount, MACFieldMainNet5DAmount, MACFieldMainNet10DAmount, MACFieldMainBuyNetAmount, MACFieldDDX, MACFieldDDY, MACFieldDDZ, MACFieldDDF, MACFieldStockFlagA, MACFieldStockFlagB, MACFieldAuctionVolRatio, MACFieldPrevAmount, MACFieldRecentIndicator, MACFieldBid3Price, MACFieldBid4Price, MACFieldBid5Price, MACFieldAsk3Price, MACFieldAsk4Price, MACFieldAsk5Price, MACFieldBid3Volume, MACFieldBid4Volume, MACFieldUpCount, MACFieldAsk3Volume, MACFieldAsk4Volume, MACFieldDownCount, MACFieldBidAskDiff, MACFieldChangeUpType, MACFieldSafetyScore, MACFieldHighlightCount, MACFieldChangeAt1000, MACFieldChangeAt1030, MACFieldChangeAt1100, MACFieldChangeAt1130, MACFieldChangeAt1330, MACFieldChangeAt1400, MACFieldChangeAt1430} {
			bit.applyMACFieldBitmap(bitmap)
		}
	}
}

// MACFieldBitmap 根据传入的字段位与预设组合构造 20 字节 MAC 动态字段位图。
//
// 参数 `selectors` 为可变长选择项，可传入单个字段位（`MACFieldBit`）或字段预设（`MACPresetField`），
// 按传入顺序依次叠加置位；值为 `nil` 的选择项会被跳过。返回独立的 20 字节位图副本，置 1 的位
// 对应请求启用的动态字段，调用方可自由修改；函数不持有共享状态，可并发调用，不返回错误。
func MACFieldBitmap(selectors ...MACFieldSelector) [20]byte {
	var bitmap [20]byte
	for _, selector := range selectors {
		// 跳过空选择项，兼容调用方混入空预设的场景，避免空指针调用。
		if selector == nil {
			continue
		}
		selector.applyMACFieldBitmap(&bitmap)
	}
	return bitmap
}

// DefaultMACBoardMembersQuotesFieldBitmap 返回成分股动态字段报价接口使用的默认字段位图。
//
// 该函数无参数；返回固定的 20 字节字面量，置 1 的位对应默认启用的动态字段，不返回错误。
// 调用方可在此基础上按需增删字段位。
func DefaultMACBoardMembersQuotesFieldBitmap() [20]byte {
	return [20]byte{
		0xff, 0xfc, 0xe1, 0xcc, 0x3f, 0x08, 0x03, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
}

// FullMACBoardMembersQuotesFieldBitmap 返回 20 字节全部置 1 的全字段位图，适合实验性
// 全字段请求。该函数无参数，返回独立位图副本，不返回错误。
func FullMACBoardMembersQuotesFieldBitmap() [20]byte {
	return MACFieldBitmap(MACPresetDebug)
}

// MACBoardMembersQuotesFieldBitmapFromBits 按传入的位号列表组装 20 字节字段位图。
//
// 参数 `bits` 为可变长位号列表，有效取值 0–159，对应 20 字节位图的各个二进制位；负数或超出
// 0–159 范围的位号会被忽略，不会产生越界写入。返回组装完成的独立位图副本，不返回错误。
func MACBoardMembersQuotesFieldBitmapFromBits(bits ...int) [20]byte {
	var bitmap [20]byte
	for _, bit := range bits {
		// 位号必须落在 20 字节位图的 0–159 范围内，越界直接跳过。
		if bit < 0 || bit >= len(bitmap)*8 {
			continue
		}
		// 按 `bit/8` 定位字节、`bit%8` 定位字节内的位，按位或完成置位。
		bitmap[bit/8] |= 1 << uint(bit%8)
	}
	return bitmap
}

// DefaultMACSymbolQuotesFieldBitmap 返回个股批量报价接口使用的默认字段位图。该函数无参数，
// 返回与成分股报价一致的默认位图副本，不返回错误。
func DefaultMACSymbolQuotesFieldBitmap() [20]byte {
	return DefaultMACBoardMembersQuotesFieldBitmap()
}

// FullMACSymbolQuotesFieldBitmap 返回个股批量报价接口使用的全字段位图。该函数无参数，
// 返回全部位置 1 的独立位图副本，不返回错误。
func FullMACSymbolQuotesFieldBitmap() [20]byte {
	return FullMACBoardMembersQuotesFieldBitmap()
}

// MACSymbolQuotesFieldBitmapFromBits 按传入的位号列表组装个股批量报价通用位图。
//
// 参数 `bits` 为可变长位号列表，有效取值 0–159，越界位号会被忽略；返回独立位图副本，不返回错误。
func MACSymbolQuotesFieldBitmapFromBits(bits ...int) [20]byte {
	return MACBoardMembersQuotesFieldBitmapFromBits(bits...)
}

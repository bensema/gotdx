package types

import (
	"errors"
	"fmt"
	"strings"
)

// Market 标识证券所属的交易市场，底层为 `uint8`，取值与协议市场编号一一对应。
//
// 该类型为无状态值类型，可按值安全传递；零值 `MarketSZ` 并非“未知市场”，未识别的市场由
// 调用方以零值之外的判断区分。`Uint8` 返回协议请求使用的数值，`String` 返回两位大写代码用于
// 拼接证券代码，`Name` 返回中文市场名。
type Market uint8

// MarketSZ 为深圳证券交易所市场，编号 0，覆盖深市 A 股、深市 ETF 与深市指数。
//
// `Market*` 市场代码枚举，自 0 起依次编号；配合 `String`、`Name` 方法完成后缀拼接与名称展示。
const (
	MarketSZ  Market = iota // 深圳市场，编号 0。
	MarketSH                // 上海市场，编号 1。
	MarketBJ                // 北京证券交易所市场，编号 2。
	MarketHK                // 香港交易所市场（港股），编号 3。
	MarketUSA               // 美股市场，编号 4。
)

// 品种类型枚举，区分股票、指数、板块与 ETF，作为行情品种分类参数使用。
const (
	STOCK = iota // 股票，编号 0。
	INDEX        // 指数，编号 1。
	BLOCK        // 板块，编号 2。
	ETF          // 交易型开放式指数基金，编号 3。
)

// MarketSZName 是深市两位大写后缀代码 `SZ`，用于把纯代码拼接为 `代码.SZ` 形式。
//
// `Market*Name` 保存各市场对应的两位大写后缀字符串，取值与 `Market.String` 的返回值一致。
const (
	MarketSZName string = "SZ" // 深市后缀代码。
	MarketSHName string = "SH" // 沪市后缀代码。
	MarketBJName string = "BJ" // 北交所后缀代码。
	MarketHKName string = "HK" // 港股后缀代码。
	MarketUSName string = "US" // 美股后缀代码。
)

// Uint8 返回 `Market` 的底层 `uint8` 数值，直接作为协议请求中的市场编号字段发送。
//
// 参数 `obj` 为当前市场值；返回值为其底层编号，零值市场返回 0。
func (obj Market) Uint8() uint8 { return uint8(obj) }

// String 返回 `Market` 对应的两位大写后缀代码字符串，用于拼接证券代码。
//
// 参数 `obj` 为当前市场值；已知市场返回 `SZ`、`SH`、`BJ`、`HK`、`US` 之一，未识别市场返回
// 字面量 `unknown` 作为兜底标记。
func (obj Market) String() string {
	switch obj {
	case MarketSZ:
		return "SZ"
	case MarketSH:
		return "SH"
	case MarketBJ:
		return "BJ"
	case MarketHK:
		return "HK"
	case MarketUSA:
		return "US"
	default:
		return "unknown"
	}
}

// Name 返回 `Market` 对应的中文市场名，用于界面展示与日志输出。
//
// 参数 `obj` 为当前市场值；已知市场返回对应中文名，未识别市场返回“未知”。
func (obj Market) Name() string {
	switch obj {
	case MarketSH:
		return "上海"
	case MarketSZ:
		return "深圳"
	case MarketBJ:
		return "北京"
	case MarketHK:
		return "香港"
	case MarketUSA:
		return "美国"
	default:
		return "未知"
	}
}

// BlockType 表示个股所属的板块分类，由 `Block` 与 `GetStockBlock` 按代码前缀判定得出。
//
// 底层为 `uint8`，取值自 0 起依次编号；判定失败或代码无法归类时返回 `BlockOther`。
type BlockType uint8

// BlockMainBoard 为沪深主板板块，由沪市 6 开头、深市 0 开头的代码归入。
//
// `BlockType*` 个股板块分类集合，覆盖主板、科创板、创业板、北证、港股、美股等，编号自 0 起。
const (
	BlockMainBoard BlockType = iota // 沪深主板，编号 0。
	BlockSTAR                       // 科创板，编号 1（沪市 688 开头）。
	BlockChiNext                    // 创业板，编号 2（深市 300、301 开头）。
	BlockBJSE                       // 北交所板块，编号 3（9 开头）。
	BlockHK                         // 港股板块，编号 4。
	BlockUS                         // 美港
	BlockOther                      // 其他板块，编号 6；无法解析或前缀不匹配时归入。
)

// Block 根据证券代码判定个股所属板块分类。
//
// 参数 `code` 可传入六位纯代码，也可传入带两位后缀的 `代码.后缀` 形式；函数先经 `DecodeStockCode`
// 解析市场与纯代码，再按市场和代码前缀细分板块。返回第一项为板块类型；代码无法解析时返回
// `BlockOther` 并携带解析错误，市场未知时返回 `BlockOther` 且错误为 `nil`。
func Block(code string) (BlockType, error) {
	market, symbol, err := DecodeStockCode(code)
	if err != nil {
		return BlockOther, err
	}
	// 板块判定只依赖代码前缀，统一拼接后缀仅为复用 `GetStockBlock` 的入参格式。
	newCode := fmt.Sprintf("%s.%s", symbol, MarketSH.String())
	switch market {
	case MarketSH:
		return GetStockBlock(newCode), nil
	case MarketSZ:
		return GetStockBlock(newCode), nil
	case MarketBJ:
		return GetStockBlock(newCode), nil
	case MarketHK:
		return BlockHK, nil
	case MarketUSA:
		return BlockUS, nil
	default:
		return BlockOther, nil
	}
}

// GetStockBlock 根据六位股票代码判定所属 A 股板块。
//
// 参数 `code` 为六位数字代码（如 `600000`、`688001`、`300750`、`920001`、`830799`），也允许
// 带 `.SH`、`.SZ`、`.BJ` 后缀，函数先剥离后缀再取前缀判定。非六位代码归入 `BlockOther`；
// 其余按前缀判定科创板、创业板、北证、主板，前缀均不匹配者归入 `BlockOther`，函数不返回错误。
func GetStockBlock(code string) BlockType {
	// 统一去除常见交易所后缀，仅保留代码段参与前缀判定。
	code = strings.TrimSuffix(code, ".SH")
	code = strings.TrimSuffix(code, ".SZ")
	code = strings.TrimSuffix(code, ".BJ")

	// 长度必须为六位，否则前缀判定不可靠，归入其他板块。
	if len(code) != 6 {
		return BlockOther
	}

	prefix3 := code[:3]
	prefix1 := code[:1]
	switch {
	// 科创板：688 开头。
	case prefix3 == "688":
		return BlockSTAR
	// 创业板：300、301 开头。
	case prefix1 == "3":
		return BlockChiNext
	case prefix1 == "9":
		return BlockBJSE
	case prefix1 == "0" || prefix1 == "6":
		return BlockMainBoard
	default:
		return BlockOther
	}
}

// ExMarketStock 为扩展行情中的境内股票品种类别，编号 1。
//
// `ExMarket*` 扩展行情协议的大类品种与市场代码，作为扩展行情请求的品种类别参数传入；编号与
// 服务端品种类型一一对应，映射关系见 `exGoodsTypeNames` 对照表。
const (
	ExMarketStock        uint8 = 1  // 境内股票（A 股等）。
	ExMarketHK           uint8 = 2  // 港股品种。
	ExMarketFutures      uint8 = 3  // 境内外期货品种。
	ExMarketFX           uint8 = 4  // 外汇（汇率）品种。
	ExMarketIndex        uint8 = 5  // 指数品种。
	ExMarketValuation    uint8 = 6  // 品种估值数据，细分含义语义待确认。
	ExMarketMoney        uint8 = 7  // 资金（资金流向）品种，细分含义语义待确认。
	ExMarketFund         uint8 = 8  // 基金品种。
	ExMarketMonetaryFund uint8 = 9  // 货币基金品种。
	ExMarketIndicator    uint8 = 10 // 指标（宏观或技术指标），细分含义语义待确认。
	ExMarketMirror       uint8 = 11 // 镜像品种，用途语义待确认。
	ExMarketOption       uint8 = 12 // 期权品种。
	ExMarketUS           uint8 = 13 // 美股品种。
	ExMarketDE           uint8 = 14 // 德国市场品种，细分含义语义待确认。
	ExMarketSG           uint8 = 15 // 新加坡市场品种，细分含义语义待确认。
)

// ExCategoryTempStock 为编号 1 的扩展分类，名称指向临时股票，具体归属语义待确认。
//
// `ExCategory*` 扩展市场细分类别代码，作为扩展行情请求的品类参数使用；名称前缀对应交易所或品种
// 类别（郑州、大连、上海、中金所、COMEX、NYMEX、CBOT 等），无法由名称确证的编号标注“语义待确认”。
const (
	ExCategoryTempStock           uint8 = 1   // 临时股票，语义待确认。
	ExCategoryZZFuturesOption     uint8 = 4   // 郑州商品交易所期货期权。
	ExCategoryDLFuturesOption     uint8 = 5   // 大连商品交易所期货期权。
	ExCategorySHFuturesOption     uint8 = 6   // 上海期货交易所期货期权。
	ExCategoryCFFEXOption         uint8 = 7   // 中国金融期货交易所期权。
	ExCategorySHStockOption       uint8 = 8   // 沪市股票期权。
	ExCategorySZStockOption       uint8 = 9   // 深市股票期权。
	ExCategoryBasicFX             uint8 = 10  // 基础外汇（汇率）品种。
	ExCategoryCrossFX             uint8 = 11  // 交叉汇率品种，具体配对语义待确认。
	ExCategoryIntlIndex           uint8 = 12  // 国际指数品种。
	ExCategoryCOMEXFutures        uint8 = 16  // COMEX（纽约商品交易所）期货。
	ExCategoryNYMEXFutures        uint8 = 17  // NYMEX（纽约商业交易所）期货。
	ExCategoryCBOTFutures         uint8 = 18  // CBOT（芝加哥期货交易所）期货。
	ExCategoryHKFinancialFutures  uint8 = 23  // 港股金融期货，具体标的语义待确认。
	ExCategoryHKFinancialOptions  uint8 = 24  // 港股金融期权，具体标的语义待确认。
	ExCategoryHKStockFutures      uint8 = 25  // 港股股票期货。
	ExCategoryHKStockOptions      uint8 = 26  // 港股股票期权。
	ExCategoryHKIndex             uint8 = 27  // 港股指数。
	ExCategoryZZFutures           uint8 = 28  // 郑州商品交易所期货。
	ExCategoryDLFutures           uint8 = 29  // 大连商品交易所期货。
	ExCategorySHFutures           uint8 = 30  // 上海期货交易所期货。
	ExCategoryHKMainBoard         uint8 = 31  // 港股主板品种。
	ExCategoryOpenEndFund         uint8 = 33  // 开放式基金。
	ExCategoryMonetaryFund        uint8 = 34  // 货币基金。
	ExCategoryMacroIndicator      uint8 = 38  // 宏观经济指标。
	ExCategoryFuturesIndex        uint8 = 42  // 期货指数，构成方式语义待确认。
	ExCategoryBToH                uint8 = 43  // B 股转 H 股相关分类，语义待确认。
	ExCategoryNEEQ                uint8 = 44  // 全国中小企业股份转让系统（新三板）品种。
	ExCategorySHGold              uint8 = 46  // 上海黄金品种。
	ExCategoryCFFEXFutures        uint8 = 47  // 中国金融期货交易所期货。
	ExCategoryHKGEM               uint8 = 48  // 港股创业板（GEM 板）。
	ExCategoryHKFund              uint8 = 49  // 港股基金。
	ExCategoryTreasuryValuation   uint8 = 54  // 国债估值数据，计算口径语义待确认。
	ExCategorySunshinePrivateFund uint8 = 56  // 阳光私募基金。
	ExCategoryBrokerCollective    uint8 = 57  // 券商集合理财产品。
	ExCategoryBrokerMonetary      uint8 = 58  // 券商货币类产品，细分语义待确认。
	ExCategoryMainFuturesContract uint8 = 60  // 期货主力合约。
	ExCategoryCSIIndex            uint8 = 62  // 中证指数系列。
	ExCategoryGZArbitrageFutures  uint8 = 65  // 国债套利期货，套利结构语义待确认。
	ExCategoryGZFutures           uint8 = 66  // 国债期货。
	ExCategoryGZOptions           uint8 = 67  // 国债期权，品种范围语义待确认。
	ExCategoryRiskControlIndex    uint8 = 68  // 风控指数，编制规则语义待确认。
	ExCategoryHuazhengIndex       uint8 = 69  // 华证指数系列。
	ExCategoryExtendedSectorIndex uint8 = 70  // 扩展板块指数，构成方式语义待确认。
	ExCategoryHKStock             uint8 = 71  // 港股个股。
	ExCategoryGEStock             uint8 = 73  // 德国股票品种，语义待确认。
	ExCategoryUSStock             uint8 = 74  // 美股个股。
	ExCategorySGStock             uint8 = 78  // 新加坡股票品种。
	ExCategoryMoneyMarket         uint8 = 91  // 货币市场品种，具体标的语义待确认。
	ExCategoryFundValuation       uint8 = 93  // 基金估值数据，计算口径语义待确认。
	ExCategoryHKDarkPool          uint8 = 98  // 港股暗池交易分类，语义待确认。
	ExCategoryCodeMirror          uint8 = 100 // 代码镜像分类，用途语义待确认。
	ExCategorySZSEIndex           uint8 = 102 // 深证指数系列。
)

// DefaultSecurityListCount 为证券列表接口默认一次拉取的条目条数，取值 1600。
//
// `Default*` 各行情接口在调用方未显式指定条数、数量或大小时回填使用的默认值；接口参数传 0 或
// 省略时按对应默认值发送协议请求。
const (
	DefaultSecurityListCount = 1600   // `GetSecurityList` 默认拉取条数。
	DefaultTickChartCount    = 0xba00 // `GetMinuteTimeData`、`GetTickChart` 默认分时请求点数。
	DefaultAuctionCount      = 500    // 集合竞价记录默认请求条数。
	DefaultUnusualCount      = 600    // 异动监控记录默认请求条数。
	DefaultTopBoardSize      = 20     // 排行榜每个榜单默认返回的条目数量。
	DefaultQuotesListCount   = 80     // 主行情快照列表默认请求条数。
	DefaultExListCount       = 2000   // 扩展市场标的列表默认条数，`count` 传 0 时回填。
	DefaultExQuotesCount     = 100    // 扩展市场行情列表默认条数，`count` 传 0 时回填。
	DefaultDownloadSize      = 0x7530 // 文件下载单块默认字节大小。
)

// AdjustNone 为不复权方式，编号 0，K 线价格按除权除息后的原始成交价展示。
//
// `Adjust*` K 线复权方式，作为 K 线请求的复权参数传入；编号 0 不复权、1 前复权、2 后复权。
const (
	AdjustNone uint16 = 0 // 不复权。
	AdjustQFQ  uint16 = 1 // 前复权：以最新价为基准向前调整历史价格。
	AdjustHFQ  uint16 = 2 // 后复权：以上市首日价为基准向后调整最新及历史价格。
)

// CategorySH 为上证 A 股列表分类，编号 0。
//
// `Category*` 主行情证券列表分类代码，用于按市场或板块拉取证券列表；编号为协议约定的分类编号。
const (
	CategorySH  uint8 = 0  // 上证 A 股。
	CategorySZ  uint8 = 2  // 深证 A 股。
	CategoryA   uint8 = 6  // 全部 A 股。
	CategoryB   uint8 = 7  // B 股。
	CategoryKCB uint8 = 8  // 科创板。
	CategoryBJ  uint8 = 12 // 北证 A 股。
	CategoryCYB uint8 = 14 // 创业板。
)

// FilterNew 为排除新股的过滤位，位值 1（二进制最低位）。
//
// `Filter*` 行情列表筛选条件位图，每个条件独占一个二进制位；多个条件按位或组合后写入请求过滤
// 字段，命中条件的标的将被排除，位值依次为 1、2、4、8、16、32、64、128。
const (
	FilterNew          uint16 = 1   // 排除新上市标的。
	FilterKC           uint16 = 2   // 排除科创板标的。
	FilterST           uint16 = 4   // 排除 ST 风险警示股。
	FilterCY           uint16 = 8   // 排除创业板标的。
	FilterHKConnect    uint16 = 16  // 排除港股通标的。
	FilterBJ           uint16 = 32  // 排除北交所标的。
	FilterApproval     uint16 = 64  // 排除核准制发行标的。
	FilterRegistration uint16 = 128 // 排除注册制发行标的。
)

// SortCode 为按证券代码排序的字段编号，取值 0x0。
//
// `Sort*` 行情列表排序字段代码，指定行情榜单按哪一个字段排序；字段语义按常量名称前缀对应行情
// 快照中的报价、成交、涨跌、资金与估值字段，编号为协议约定的排序字段代码。
const (
	SortCode              uint16 = 0x0   // 按证券代码排序。
	SortName              uint16 = 0x1   // 按证券名称排序。
	SortPreClose          uint16 = 0x2   // 按昨收价排序。
	SortOpen              uint16 = 0x3   // 按今开价排序。
	SortHigh              uint16 = 0x4   // 按最高价排序。
	SortLow               uint16 = 0x5   // 按最低价排序。
	SortPrice             uint16 = 0x6   // 按最新价排序。
	SortBid               uint16 = 0x7   // 按买一价排序。
	SortAsk               uint16 = 0x8   // 按卖一价排序。
	SortVolume            uint16 = 0x9   // 按成交量排序。
	SortTotalAmount       uint16 = 0x0a  // 按成交额排序。
	SortLastVolume        uint16 = 0x0b  // 按现量（最近一笔成交量）排序。
	SortChange            uint16 = 0x0c  // 按涨跌额排序。
	SortIndustry          uint16 = 0x0d  // 按所属行业排序。
	SortChangePct         uint16 = 0x0e  // 按涨跌幅排序。
	SortAmplitudePct      uint16 = 0x0f  // 按振幅排序。
	SortAvg               uint16 = 0x10  // 按成交均价排序。
	SortPEDynamic         uint16 = 0x11  // 按动态市盈率排序。
	SortEntrustRatio      uint16 = 0x12  // 按委比排序。
	SortInsideVolume      uint16 = 0x13  // 按内盘成交量排序。
	SortOutsideVolume     uint16 = 0x14  // 按外盘成交量排序。
	SortInOutRatio        uint16 = 0x15  // 按内外盘比值排序，计算口径语义待确认。
	SortBidVolume         uint16 = 0x17  // 按买盘挂单量排序，具体档位语义待确认。
	SortAskVolume         uint16 = 0x18  // 按卖盘挂单量排序，具体档位语义待确认。
	SortLockedRatio       uint16 = 0x1b  // 按锁定比例排序，锁定口径语义待确认。
	SortLockedAmount      uint16 = 0x1c  // 按锁定金额排序，锁定口径语义待确认。
	SortOpenAmount        uint16 = 0x1d  // 按开盘成交金额排序，语义待确认。
	SortOpenTurnoverPct   uint16 = 0x1e  // 按开盘换手比例排序，计算口径语义待确认。
	SortVolRatio          uint16 = 0x23  // 按量比排序。
	SortTurnoverRate      uint16 = 0x24  // 按换手率排序。
	SortFloatShares       uint16 = 0x25  // 按流通股本排序。
	SortFloatMarketCap    uint16 = 0x26  // 按流通市值排序。
	SortTotalMarketCapAB  uint16 = 0x27  // 按 A、B 股合计总市值排序。
	SortUnmatchedVolume   uint16 = 0x2a  // 按未匹配量排序，场景语义待确认。
	SortStrengthPct       uint16 = 0x2d  // 按强弱度排序，指标算法语义待确认。
	SortSpeedPct          uint16 = 0x2e  // 按涨速排序，统计窗口语义待确认。
	SortActivity          uint16 = 0x2f  // 按活跃度排序，指标算法语义待确认。
	SortShortTurnoverPct  uint16 = 0xcc  // 按短周期换手率排序。
	SortVolSpeedPct       uint16 = 0xd0  // 按量涨速排序。
	SortChange5DPct       uint16 = 0xd1  // 按 5 日涨跌幅排序。
	SortChange10DPct      uint16 = 0xd2  // 按 10 日涨跌幅排序。
	SortMainNetAmount     uint16 = 0xd4  // 按主力净流入金额排序。
	SortMainNetRatio      uint16 = 0xd7  // 按主力净流入占比排序。
	SortAuctionLimitBuy   uint16 = 0x102 // 按竞价买入上限相关字段排序，语义待确认。
	SortAuctionLimitSell  uint16 = 0x103 // 按竞价卖出下限相关字段排序，语义待确认。
	SortLimitUpTime       uint16 = 0x108 // 按触及涨停时间排序。
	SortLimitDownTime     uint16 = 0x109 // 按触及跌停时间排序。
	SortOpenSnatchPct     uint16 = 0x10a // 按开盘抢筹比例排序，指标算法语义待确认。
	SortAmount2M          uint16 = 0x10c // 按两分钟成交额排序，统计窗口语义待确认。
	SortOpenPct           uint16 = 0x119 // 按开盘涨幅排序，计算口径语义待确认。
	SortHighPct           uint16 = 0x11a // 按最高涨幅排序，计算口径语义待确认。
	SortLowPct            uint16 = 0x11b // 按最低涨幅排序，计算口径语义待确认。
	SortAvgChangePct      uint16 = 0x11c // 按均价涨幅排序，计算口径语义待确认。
	SortDrawdownPct       uint16 = 0x11e // 按回撤幅度排序。
	SortAttackPct         uint16 = 0x11f // 按攻击幅度排序，指标算法语义待确认。
	SortYTDPct            uint16 = 0x121 // 按年初至今涨幅排序。
	SortConsecutiveUpDays uint16 = 0x138 // 按连续上涨天数排序。
	SortChange3DPct       uint16 = 0x139 // 按 3 日涨跌幅排序。
	SortDividendYieldRate uint16 = 0x13a // 按股息率排序。
	SortChange20DPct      uint16 = 0x172 // 按 20 日涨跌幅排序。
	SortChange60DPct      uint16 = 0x173 // 按 60 日涨跌幅排序。
	SortPrevChangePct     uint16 = 0x17e // 按上一交易日涨跌幅排序。
	SortMTDPct            uint16 = 0x19c // 按月初至今涨幅排序。
	SortChange1YPct       uint16 = 0x19d // 按一年涨跌幅排序。
	SortPETTM             uint16 = 0x3ee // 按滚动市盈率（PE-TTM）排序。
	SortPEStatic          uint16 = 0x3ef // 按静态市盈率排序。
)

// SortOrderNone 为不指定排序方向，编号 0，方向由协议默认规则决定。
//
// `SortOrder*` 行情列表排序方向编号，作为排序方向参数传入：0 不指定、1 降序、2 升序。
const (
	SortOrderNone uint16 = 0 // 不指定排序方向。
	SortOrderDesc uint16 = 1 // 降序排列。
	SortOrderAsc  uint16 = 2 // 升序排列。
)

// BlockFileDefault 为服务端默认板块数据文件名 `block.dat`。
//
// `BlockFile*` 通达信服务端板块数据文件名，用于板块文件下载与解析，依次对应默认板块、地区板块、
// 风格板块与概念板块。
const (
	BlockFileDefault = "block.dat"    // 默认板块数据文件。
	BlockFileZS      = "block_zs.dat" // 地区板块数据文件。
	BlockFileFG      = "block_fg.dat" // 风格板块数据文件。
	BlockFileGN      = "block_gn.dat" // 概念板块数据文件。
)

// BoardTypeHY 为行业板块类型，编号 0。
//
// `BoardType*` 板块类型代码，用于板块列表与 MAC 板块接口的板块分类参数；已知取值覆盖行业、概念、
// 风格、地区与全部板块，无法由协议文档确证的细分编号标注“语义待确认”。
const (
	BoardTypeHY      uint16 = 0   // 行业板块。
	BoardTypeHY2     uint16 = 1   // 行业二级分类，细分口径语义待确认。
	BoardTypeGN      uint16 = 3   // 概念板块。
	BoardTypeFG      uint16 = 4   // 风格板块。
	BoardTypeDQ      uint16 = 5   // 地区板块。
	BoardTypeUnknown uint16 = 6   // 未知板块类型，语义待确认。
	BoardTypeHYOther uint16 = 7   // 其他行业板块。
	BoardTypeOther11 uint16 = 8   // 其他细分板块类型，语义待确认。
	BoardTypeOther12 uint16 = 9   // 其他细分板块类型，语义待确认。
	BoardTypeAll     uint16 = 255 // 全部板块类型。
)

// ExBoardTypeHKAll 为港股全部板块类型，编号 0。
//
// `ExBoardType*` 扩展市场（港股、美股）板块类型代码，分别对应全部、概念与行业板块，编号自 0 起
// 依次排列。
const (
	ExBoardTypeHKAll uint16 = 0 // 港股全部板块。
	ExBoardTypeHKGN  uint16 = 1 // 港股概念板块。
	ExBoardTypeHKHY  uint16 = 2 // 港股行业板块。
	ExBoardTypeUSAll uint16 = 3 // 美股全部板块。
	ExBoardTypeUSGN  uint16 = 4 // 美股概念板块。
	ExBoardTypeUSHY  uint16 = 5 // 美股行业板块。
)

// KLINE_TYPE_5MIN 为 5 分钟周期 K 线类别，协议编号 0。
//
// KLINE_TYPE_* K 线周期类型代码，作为 K 线请求的周期参数 `Category` 使用；编号与服务端周期约定
// 一一对应，分钟类周期的时间戳按位压缩日期加当日累计分钟解码，日及以上周期按打包年月日解码。
const (
	KLINE_TYPE_5MIN      = 0  // 5 分钟 K 线，编号 0。
	KLINE_TYPE_15MIN     = 1  // 15 分钟 K 线，编号 1。
	KLINE_TYPE_30MIN     = 2  // 30 分钟 K 线，编号 2。
	KLINE_TYPE_1HOUR     = 3  // 1 小时 K 线，编号 3。
	KLINE_TYPE_DAILY     = 4  // 日 K 线，编号 4。
	KLINE_TYPE_WEEKLY    = 5  // 周 K 线，编号 5。
	KLINE_TYPE_MONTHLY   = 6  // 月 K 线，编号 6。
	KLINE_TYPE_EXHQ_1MIN = 7  // 扩展行情（EXHQ）1 分钟 K 线，编号 7。
	KLINE_TYPE_1MIN      = 8  // 1 分钟 K 线，编号 8。
	KLINE_TYPE_RI_K      = 9  // 日 K 线；另一套日线编码，与编号 4 的时间打包方式不同，语义待确认。
	KLINE_TYPE_3MONTH    = 10 // 季 K 线，编号 10。
	KLINE_TYPE_YEARLY    = 11 // 年 K 线，编号 11。
	KLINE_TYPE_SECONDS   = 13 // 多秒 K 线，编号 13，时间戳按秒级格式解码。
)

// ErrBadData 是服务端返回数据超过协议可承受大小（8M 以上）时抛出的错误。
//
// 错误消息为协议原始英文文案，保持原样不改写；调用方通过错误是否非空判断响应数据量超限。
var (
	ErrBadData = errors.New("more than 8M data")
)

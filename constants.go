package gotdx

import "errors"

const (
	MarketSZ uint8 = 0 // 深圳
	MarketSH uint8 = 1 // 上海
	MarketBJ uint8 = 2 // 北京

	// Backward-compatible aliases.
	MarketSz = MarketSZ
	MarketSh = MarketSH
	MarketBj = MarketBJ
)

const (
	ExMarketStock        uint8 = 1  // 股票
	ExMarketHK           uint8 = 2  // 香港
	ExMarketFutures      uint8 = 3  // 期货
	ExMarketFX           uint8 = 4  // 汇率
	ExMarketIndex        uint8 = 5  // 指数
	ExMarketValuation    uint8 = 6  // 估值
	ExMarketMoney        uint8 = 7  // 资金
	ExMarketFund         uint8 = 8  // 基金
	ExMarketMonetaryFund uint8 = 9  // 货币基金
	ExMarketIndicator    uint8 = 10 // 指标
	ExMarketMirror       uint8 = 11 // 镜像
	ExMarketOption       uint8 = 12 // 期权
	ExMarketUS           uint8 = 13 // 美国
	ExMarketDE           uint8 = 14 // 德国
	ExMarketSG           uint8 = 15 // 新加坡
)

const (
	ExCategoryTempStock           uint8 = 1
	ExCategoryZZFuturesOption     uint8 = 4
	ExCategoryDLFuturesOption     uint8 = 5
	ExCategorySHFuturesOption     uint8 = 6
	ExCategoryCFFEXOption         uint8 = 7
	ExCategorySHStockOption       uint8 = 8
	ExCategorySZStockOption       uint8 = 9
	ExCategoryBasicFX             uint8 = 10
	ExCategoryCrossFX             uint8 = 11
	ExCategoryIntlIndex           uint8 = 12
	ExCategoryCOMEXFutures        uint8 = 16
	ExCategoryNYMEXFutures        uint8 = 17
	ExCategoryCBOTFutures         uint8 = 18
	ExCategoryHKFinancialFutures  uint8 = 23
	ExCategoryHKFinancialOptions  uint8 = 24
	ExCategoryHKStockFutures      uint8 = 25
	ExCategoryHKStockOptions      uint8 = 26
	ExCategoryHKIndex             uint8 = 27
	ExCategoryZZFutures           uint8 = 28
	ExCategoryDLFutures           uint8 = 29
	ExCategorySHFutures           uint8 = 30
	ExCategoryHKMainBoard         uint8 = 31
	ExCategoryOpenEndFund         uint8 = 33
	ExCategoryMonetaryFund        uint8 = 34
	ExCategoryMacroIndicator      uint8 = 38
	ExCategoryFuturesIndex        uint8 = 42
	ExCategoryBToH                uint8 = 43
	ExCategoryNEEQ                uint8 = 44
	ExCategorySHGold              uint8 = 46
	ExCategoryCFFEXFutures        uint8 = 47
	ExCategoryHKGEM               uint8 = 48
	ExCategoryHKFund              uint8 = 49
	ExCategoryTreasuryValuation   uint8 = 54
	ExCategorySunshinePrivateFund uint8 = 56
	ExCategoryBrokerCollective    uint8 = 57
	ExCategoryBrokerMonetary      uint8 = 58
	ExCategoryMainFuturesContract uint8 = 60
	ExCategoryCSIIndex            uint8 = 62
	ExCategoryGZArbitrageFutures  uint8 = 65
	ExCategoryGZFutures           uint8 = 66
	ExCategoryGZOptions           uint8 = 67
	ExCategoryRiskControlIndex    uint8 = 68
	ExCategoryHuazhengIndex       uint8 = 69
	ExCategoryExtendedSectorIndex uint8 = 70
	ExCategoryHKStock             uint8 = 71
	ExCategoryGEStock             uint8 = 73
	ExCategoryUSStock             uint8 = 74
	ExCategorySGStock             uint8 = 78
	ExCategoryMoneyMarket         uint8 = 91
	ExCategoryFundValuation       uint8 = 93
	ExCategoryHKDarkPool          uint8 = 98
	ExCategoryCodeMirror          uint8 = 100
	ExCategorySZSEIndex           uint8 = 102
)

const (
	DefaultSecurityListCount = 1600
	DefaultTickChartCount    = 0xba00
	DefaultAuctionCount      = 500
	DefaultUnusualCount      = 600
	DefaultTopBoardSize      = 20
	DefaultQuotesListCount   = 80
	DefaultExListCount       = 2000
	DefaultExQuotesCount     = 100
	DefaultDownloadSize      = 0x7530
)

const (
	AdjustNone uint16 = 0
	AdjustQFQ  uint16 = 1
	AdjustHFQ  uint16 = 2
)

const (
	CategorySH  uint8 = 0  // 上证A
	CategorySZ  uint8 = 2  // 深证A
	CategoryA   uint8 = 6  // A股
	CategoryB   uint8 = 7  // B股
	CategoryKCB uint8 = 8  // 科创板
	CategoryBJ  uint8 = 12 // 北证A
	CategoryCYB uint8 = 14 // 创业板
)

const (
	FilterNew uint16 = 1
	FilterKC  uint16 = 2
	FilterST  uint16 = 4
	FilterCY  uint16 = 8
	FilterBJ  uint16 = 16
)

const (
	SortCode             uint16 = 0x0
	SortName             uint16 = 0x1
	SortPreClose         uint16 = 0x2
	SortOpen             uint16 = 0x3
	SortHigh             uint16 = 0x4
	SortLow              uint16 = 0x5
	SortPrice            uint16 = 0x6
	SortBid              uint16 = 0x7
	SortAsk              uint16 = 0x8
	SortVolume           uint16 = 0x9
	SortTotalAmount      uint16 = 0x0a
	SortLastVolume       uint16 = 0x0b
	SortChange           uint16 = 0x0c
	SortChangePct        uint16 = 0x0e
	SortAmplitudePct     uint16 = 0x0f
	SortAvg              uint16 = 0x10
	SortPEDynamic        uint16 = 0x11
	SortEntrustRatio     uint16 = 0x12
	SortInsideVolume     uint16 = 0x13
	SortOutsideVolume    uint16 = 0x14
	SortInOutRatio       uint16 = 0x15
	SortBidVolume        uint16 = 0x17
	SortAskVolume        uint16 = 0x18
	SortLockedRatio      uint16 = 0x1b
	SortLockedAmount     uint16 = 0x1c
	SortOpenAmount       uint16 = 0x1d
	SortOpenTurnoverPct  uint16 = 0x1e
	SortVolRatio         uint16 = 0x23
	SortTurnoverRate     uint16 = 0x24
	SortFloatShares      uint16 = 0x25
	SortFloatMarketCap   uint16 = 0x26
	SortTotalMarketCapAB uint16 = 0x27
	SortUnmatchedVolume  uint16 = 0x2a
	SortStrengthPct      uint16 = 0x2d
	SortSpeedPct         uint16 = 0x2e
	SortActivity         uint16 = 0x2f
	SortShortTurnoverPct uint16 = 0xcc
	SortVolSpeedPct      uint16 = 0xd0
	SortMainNetAmount    uint16 = 0xd4
	SortMainNetRatio     uint16 = 0xd7
	SortAuctionLimitBuy  uint16 = 0x102
	SortOpenSnatchPct    uint16 = 0x10a
	SortAmount2M         uint16 = 0x10c
	SortOpenPct          uint16 = 0x119
	SortHighPct          uint16 = 0x11a
	SortLowPct           uint16 = 0x11b
	SortAvgChangePct     uint16 = 0x11c
	SortDrawdownPct      uint16 = 0x11e
	SortAttackPct        uint16 = 0x11f
)

const (
	SortOrderNone uint16 = 0
	SortOrderDesc uint16 = 1
	SortOrderAsc  uint16 = 2
)

const (
	BlockFileDefault = "block.dat"
	BlockFileZS      = "block_zs.dat"
	BlockFileFG      = "block_fg.dat"
	BlockFileGN      = "block_gn.dat"
)

const (
	BoardTypeHY      uint16 = 0
	BoardTypeHY2     uint16 = 1
	BoardTypeGN      uint16 = 3
	BoardTypeFG      uint16 = 4
	BoardTypeDQ      uint16 = 5
	BoardTypeUnknown uint16 = 6
	BoardTypeHYOther uint16 = 7
	BoardTypeOther11 uint16 = 8
	BoardTypeOther12 uint16 = 9
	BoardTypeAll     uint16 = 255
)

const (
	ExBoardTypeHKAll uint16 = 0
	ExBoardTypeHKGN  uint16 = 1
	ExBoardTypeHKHY  uint16 = 2
	ExBoardTypeUSAll uint16 = 3
	ExBoardTypeUSGN  uint16 = 4
	ExBoardTypeUSHY  uint16 = 5
)

const (
	KLINE_TYPE_5MIN      = 0  // 5 分钟K 线
	KLINE_TYPE_15MIN     = 1  // 15 分钟K 线
	KLINE_TYPE_30MIN     = 2  // 30 分钟K 线
	KLINE_TYPE_1HOUR     = 3  // 1 小时K 线
	KLINE_TYPE_DAILY     = 4  // 日K 线
	KLINE_TYPE_WEEKLY    = 5  // 周K 线
	KLINE_TYPE_MONTHLY   = 6  // 月K 线
	KLINE_TYPE_EXHQ_1MIN = 7  //  1 分钟
	KLINE_TYPE_1MIN      = 8  // 1 分钟K 线
	KLINE_TYPE_RI_K      = 9  // 日K 线
	KLINE_TYPE_3MONTH    = 10 // 季K 线
	KLINE_TYPE_YEARLY    = 11 // 年K 线
	KLINE_TYPE_SECONDS   = 13 // 多秒K 线
)

var (
	ErrBadData = errors.New("more than 8M data")
)

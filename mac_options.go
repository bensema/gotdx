package gotdx

// MACSortType 是 MAC 板块成分榜单的排序字段编号，作为请求的排序字段参数传入，指定结果按
// 哪一项指标排序；取值为协议约定的排序字段代码，编号定义见 `MACSort*` 常量。取值 0 表示不
// 指定排序字段，服务端默认回填 14（`MACSortChangePct`，按涨跌幅排序）。
type MACSortType uint16

// MACSort* 板块成分排序字段编号，作为请求的排序字段参数传入，指定榜单按哪一项指标排序。
// 各编号对应的排序维度见常量行尾注释；请求排序字段传 0 时服务端默认回填 `MACSortChangePct`。
const (
	MACSortCode              MACSortType = 0x00  // 按证券代码排序
	MACSortName              MACSortType = 0x01  // 按证券名称排序
	MACSortPrice             MACSortType = 0x06  // 按最新价排序
	MACSortVol               MACSortType = 0x07  // 按成交量排序
	MACSortAmount            MACSortType = 0x08  // 按成交额排序
	MACSortTotalAmount       MACSortType = 0x0a  // 按总成交额排序
	MACSortLastVolume        MACSortType = 0x0b  // 按现量（最新一笔成交量）排序
	MACSortChange            MACSortType = 0x0c  // 按涨跌额排序
	MACSortIndustry          MACSortType = 0x0d  // 按所属行业排序
	MACSortChangePct         MACSortType = 0x0e  // 按涨跌幅排序；排序字段传 0 时服务端默认回填此编号
	MACSortAmplitudePct      MACSortType = 0x0f  // 按振幅排序
	MACSortShortTurnoverPct  MACSortType = 0xcc  // 按短周期换手率排序
	MACSortVolSpeedPct       MACSortType = 0xd0  // 按量涨速排序
	MACSortChange5DPct       MACSortType = 0xd1  // 按近 5 日涨幅排序
	MACSortChange10DPct      MACSortType = 0xd2  // 按近 10 日涨幅排序
	MACSortMainNetAmount     MACSortType = 0xd4  // 按主力净流入金额排序
	MACSortMainNetRatio      MACSortType = 0xd7  // 按主力净比排序
	MACSortAuctionLimitBuy   MACSortType = 0x102 // 按连续竞价买入上限价排序
	MACSortAuctionLimitSell  MACSortType = 0x103 // 按连续竞价卖出下限价排序
	MACSortLimitUpTime       MACSortType = 0x108 // 按涨停时间排序
	MACSortLimitDownTime     MACSortType = 0x109 // 按跌停时间排序
	MACSortDrawdownPct       MACSortType = 0x11e // 按回撤幅度排序
	MACSortAttackPct         MACSortType = 0x11f // 按攻击幅度排序，具体语义待确认
	MACSortYTDPct            MACSortType = 0x121 // 按年初至今涨幅排序
	MACSortConsecutiveUpDays MACSortType = 0x138 // 按连续上涨天数排序
	MACSortChange3DPct       MACSortType = 0x139 // 按近 3 日涨幅排序
	MACSortDividendYieldRate MACSortType = 0x13a // 按股息率排序
	MACSortChange20DPct      MACSortType = 0x172 // 按近 20 日涨幅排序
	MACSortChange60DPct      MACSortType = 0x173 // 按近 60 日涨幅排序
	MACSortPrevChangePct     MACSortType = 0x17e // 按昨日涨幅排序
	MACSortMTDPct            MACSortType = 0x19c // 按月初至今涨幅排序
	MACSortChange1YPct       MACSortType = 0x19d // 按近一年涨幅排序
	MACSortPETTM             MACSortType = 0x3ee // 按滚动市盈率（PE TTM）排序
	MACSortPEStatic          MACSortType = 0x3ef // 按静态市盈率排序
)

// MACSortOrder 是 MAC 板块成分榜单的排序方向编号，作为请求的排序方向参数传入，控制排序结果
// 按升序还是降序返回；编号定义见 `MACSortOrder*` 常量。取值 0 表示不指定方向，服务端默认回填
// 1（降序）。
type MACSortOrder uint8

// MACSortOrder* 排序方向编号，作为请求的排序方向参数传入；各编号方向语义见常量行尾注释。
const (
	MACSortOrderNone MACSortOrder = 0 // 不指定排序方向；请求传 0 时服务端默认回填 1（降序）
	MACSortOrderDesc MACSortOrder = 1 // 降序，排序指标值大的标的排在前面
	MACSortOrderAsc  MACSortOrder = 2 // 升序，排序指标值小的标的排在前面
)

// MACFilterType 是 MAC 板块成分的排除条件，每个条件独占一个二进制位；多个条件按位或叠加后
// 写入请求位图的过滤字节，命中条件的标的将被排除。位定义见 `MACFilter*` 常量；取值 0 表示不
// 排除任何标的。
type MACFilterType uint8

// MACFilter* 成分股排除条件位图，每个条件独占一个二进制位，可按位或叠加组合；命中任一条件的
// 标的将被排除出榜单。各条件对应的标的类别见常量行尾注释。
const (
	MACFilterNew          MACFilterType = 1   // 排除新股
	MACFilterKCB          MACFilterType = 2   // 排除科创板标的
	MACFilterST           MACFilterType = 4   // 排除风险警示（ST）标的
	MACFilterGEM          MACFilterType = 8   // 排除创业板标的
	MACFilterHKConnect    MACFilterType = 16  // 排除港股通标的
	MACFilterBJ           MACFilterType = 32  // 排除北交所标的
	MACFilterApproval     MACFilterType = 64  // 排除核准制标的
	MACFilterRegistration MACFilterType = 128 // 排除注册制标的
)

// MACFilterFlags 将多个排除条件合并为一个协议位图字节。
//
// 参数 `filters` 为可变长排除条件，可传入零个或多个 `MACFilter*` 条件；各条件本身即 2 的幂位掩码，
// 按位或叠加后得到可放入请求过滤字节的组合值。不传条件时返回 0（不排除任何标的）；函数不返回错误。
func MACFilterFlags(filters ...MACFilterType) uint8 {
	var flags uint8
	for _, filter := range filters {
		// 各条件独占一位，按位或合并，不同条件的位互不覆盖。
		flags |= uint8(filter)
	}
	return flags
}

// macBoardMembersQuotesControlExtended 是请求位图第 19 字节的扩展控制位，固定置 1，
// 与动态字段扩展请求配套使用。
const macBoardMembersQuotesControlExtended byte = 1

// MACBoardMembersQuotesRequestBitmap 在动态字段位图基础上拼装成分股动态字段请求位图：
// 第 17 字节写入合并后的排除条件掩码，第 19 字节置扩展控制位，其余字段位保持不变。
//
// 参数 `fieldBitmap` 为 20 字节动态字段位图；参数 `filters` 为可变长排除条件，可传零个或多个。
// 入参位图按值复制后修改，不会影响调用方传入的原位图；返回拼装完成的独立位图副本，不返回错误。
func MACBoardMembersQuotesRequestBitmap(fieldBitmap [20]byte, filters ...MACFilterType) [20]byte {
	// 协议约定第 17 字节存放成分股排除条件的组合掩码。
	fieldBitmap[17] = MACFilterFlags(filters...)
	// 第 19 字节按位或置固定扩展控制位，保留该字节上已有的其他位。
	fieldBitmap[19] |= macBoardMembersQuotesControlExtended
	return fieldBitmap
}

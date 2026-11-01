package gotdx

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bensema/gotdx/proto"
	_ "modernc.org/sqlite"
)

// futuresCalendarLocation 是期货交易日历与自然时间转换统一使用的时区，取值
// `Asia/Shanghai`；国内期货夜盘与交易日归属均按该时区判定。
const futuresCalendarLocation = "Asia/Shanghai"

// resolveMACFuturesDateTimes 使用期货交易日历把 MAC 期货 K 线的夜盘条目修正为自然时间。
//
// `items` 为待修正的 K 线条目切片，函数就地改写其中夜盘条目的 `DateTime` 字段，非夜盘条目
// 保持原样；`code` 为期货合约代码，用于前缀映射交易所；`calendarPath` 为本地 `SQLite`
// 交易日历数据库路径，传空串或代码前缀无法识别交易所时直接返回 `nil`，不修改任何条目。
// 同一交易日的上一交易日查询结果在本次调用内缓存复用。夜盘时段中 21 点之后的条目归属
// 上一交易日，其余夜盘秒数归属下一自然日。日历打开或 SQL 查询失败时返回包装后的错误。
func resolveMACFuturesDateTimes(items []proto.MACSymbolBar, code, calendarPath string) error {
	exchange, ok := futuresExchange(code)
	if !ok || calendarPath == "" {
		return nil
	}

	db, err := sql.Open("sqlite", calendarPath)
	if err != nil {
		return fmt.Errorf("open futures trade calendar: %w", err)
	}
	defer db.Close()

	preTradeDates := make(map[uint32]time.Time)
	for index := range items {
		seconds := items[index].RawSeconds
		if !isMACNightSession(seconds) {
			continue
		}
		preTradeDate, found := preTradeDates[items[index].RawYMD]
		if !found {
			preTradeDate, err = futuresPreTradeDate(db, exchange, items[index].RawYMD)
			if err != nil {
				return err
			}
			preTradeDates[items[index].RawYMD] = preTradeDate
		}

		// 21 点后属于上一交易日当晚夜盘；凌晨时段已跨入下一自然日。
		if seconds >= 21*60*60 {
			items[index].DateTime = combineFuturesDateTime(preTradeDate, seconds)
			continue
		}
		items[index].DateTime = combineFuturesDateTime(preTradeDate.AddDate(0, 0, 1), seconds)
	}
	return nil
}

// futuresExchange 根据期货合约代码前缀映射交易所代码。
//
// `code` 为期货合约代码，函数先去除首尾空白、转大写并剥离尾部数字，再按品种字母前缀匹配
// 交易所。识别成功时返回 `Tushare` 交易日历使用的交易所标识（`SHFE`、`INE`、`DCE`、
// `CZCE`、`CFFEX`）与 `true`；识别失败时返回空串与 `false`。
func futuresExchange(code string) (string, bool) {
	prefix := strings.TrimRight(strings.ToUpper(strings.TrimSpace(code)), "0123456789")
	switch prefix {
	case "AL", "AO", "AG", "AU", "BR", "BU", "CU", "FU", "HC", "NI", "PB", "RB", "RU", "SN", "SP", "SS", "WR", "ZN":
		return "SHFE", true
	case "BC", "EC", "LU", "NR", "SC":
		return "INE", true
	case "A", "B", "C", "CS", "EB", "EG", "FB", "I", "J", "JD", "JM", "L", "LH", "M", "P", "PG", "PP", "RR", "V", "Y":
		return "DCE", true
	case "AP", "CF", "CJ", "CY", "FG", "JR", "MA", "OI", "PF", "PK", "PM", "PR", "RM", "RS", "SA", "SF", "SM", "SR", "TA", "UR", "WH", "ZC":
		return "CZCE", true
	case "IC", "IF", "IH", "IM", "T", "TF", "TL", "TS":
		return "CFFEX", true
	default:
		return "", false
	}
}

// isMACNightSession 判断 MAC K 线条目是否落在需要交易日历修正的夜盘时段。
//
// `seconds` 为条目自当日 00:00:00 起计的秒数；取值不小于 21 点（21×3600）或不大于 5 点
// （5×3600）时判定为夜盘，返回 `true`，其余白天时段返回 `false`。
func isMACNightSession(seconds uint32) bool {
	return seconds >= 21*60*60 || seconds <= 5*60*60
}

// futuresPreTradeDate 查询指定期货交易日对应的上一交易日。
//
// `db` 为已打开的交易日历数据库连接；`exchange` 为 `futuresExchange` 映射出的交易所标识；
// `tradeDate` 为待查询交易日，按 `YYYYMMDD` 八位正整数传入。返回上一交易日在上海时区的
// 自然日零点；日历中查不到记录或 SQL 执行失败时返回包装后的查询错误。
func futuresPreTradeDate(db *sql.DB, exchange string, tradeDate uint32) (time.Time, error) {
	var value string
	// 交易日历中 pretrade_date 即上一自然交易日，直接取该交易日自身的记录。
	if err := db.QueryRow(`
		SELECT pretrade_date
		FROM futures_trade_calendar
		WHERE exchange = ? AND cal_date = ? AND is_open = 1`, exchange, fmt.Sprintf("%08d", tradeDate)).Scan(&value); err != nil {
		return time.Time{}, fmt.Errorf("find %s pretrade_date for %08d: %w", exchange, tradeDate, err)
	}
	return parseCalendarDate(value)
}

// combineFuturesDateTime 把日历自然日与 MAC 条目中的当日秒数组合为带时区的时间。
//
// `date` 为归属自然日，结果沿用该时间携带的时区；`seconds` 为当日秒数，按时、分、秒直接
// 拆分写入，不做跨天进位处理。
func combineFuturesDateTime(date time.Time, seconds uint32) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), int(seconds/3600), int(seconds%3600/60), int(seconds%60), 0, date.Location())
}

// parseCalendarDate 把 `SQLite` 日历中的 `YYYYMMDD` 日期字符串解析为上海时区当日零点。
//
// `value` 必须为恰好 8 位十进制数字；长度非法、年月日数值非法，或组合出不存在日期（如
// 2 月 30 日）时返回错误。时区加载失败时同样返回包装后的错误。
func parseCalendarDate(value string) (time.Time, error) {
	if len(value) != 8 {
		return time.Time{}, fmt.Errorf("invalid calendar date %q", value)
	}
	location, err := time.LoadLocation(futuresCalendarLocation)
	if err != nil {
		return time.Time{}, fmt.Errorf("load %s: %w", futuresCalendarLocation, err)
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse calendar year %q: %w", value, err)
	}
	month, err := strconv.Atoi(value[4:6])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse calendar month %q: %w", value, err)
	}
	day, err := strconv.Atoi(value[6:])
	if err != nil {
		return time.Time{}, fmt.Errorf("parse calendar day %q: %w", value, err)
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, location)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return time.Time{}, fmt.Errorf("invalid calendar date %q", value)
	}
	return date, nil
}

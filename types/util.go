package types

import (
	"fmt"
	"regexp"
	"strings"
)

// DetectMarket 从带市场后缀的证券代码中识别所属市场，并返回市场枚举与去掉后缀的纯代码。
//
// 参数 `symbol` 为 `代码.两位后缀` 形式的证券代码（如 `000001.SZ`、`600000.SH`、`00700.HK`、
// `AAPL.US`）；首尾空白与字母大小写不敏感，函数先归一化输入格式。返回第一项为命中的市场枚举，
// 第二项为剥离点号与两位后缀后的纯代码；缺少分隔点、长度不足或后缀不属于已知交易所时，第三项返回
// 格式错误，此时市场枚举与纯代码均为空。待识别的纯代码请先经 `StockPrefix` 或 `DecodeStockCode`
// 补全后缀后再调用本函数。
func DetectMarket(symbol string) (Market, string, error) {
	// 先归一化首尾空白与大小写，保证后续点号定位与后缀比较不受输入格式影响。
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	// 末尾三位应为“点号 + 两位后缀”，先取该段校验分隔点是否存在。
	prefix := symbol[len(symbol)-3:]
	if prefix[0] != '.' {
		return 0, "", fmt.Errorf("股票代码错误,例如:000001.SZ")
	}
	// 点号之前为纯代码段，最后两位为市场后缀，分别取出供后缀匹配。
	code := symbol[:len(symbol)-3]
	market := symbol[len(symbol)-2:]
	switch market {
	case MarketSH.String():
		return MarketSH, code, nil
	case MarketSZ.String():
		return MarketSZ, code, nil
	case MarketBJ.String():
		return MarketBJ, code, nil
	case MarketHK.String():
		return MarketHK, code, nil
	case MarketUSA.String():
		return MarketUSA, code, nil
	default:
		return 0, "", fmt.Errorf("股票代码错误,例如:000001.SZ")
	}
}

// DecodeStockCode 同时兼容纯代码与带后缀代码，解析出市场枚举与纯证券代码。
//
// 参数 `code` 既可传六位纯代码，也可传携带旧后缀的代码；函数先用 `CleanCode` 剥离旧后缀，再按
// `StockPrefix` 的代码段规则补全标准后缀，最后交由 `DetectMarket` 识别市场。返回第二项为剥离后缀
// 的纯代码；代码段规则无法判定任何已知市场时返回格式错误，市场枚举与纯代码为空。
//
// 与 `DetectMarket` 的区别在于本函数会先补全后缀，可直接接受未带后缀的纯代码输入。
func DecodeStockCode(code string) (Market, string, error) {
	code = StockPrefix(CleanCode(code))
	return DetectMarket(code)
}

// StockPrefix 根据纯代码规则为股票、ETF、指数代码补全交易所后缀，返回 `代码.后缀`；无法识别时原样返回纯代码。
//
// 参数 `code` 为待补全的纯代码（允许携带旧后缀，函数先经 `CleanCode` 清理）；判定自上而下短路匹配，
// 命中即返回，优先级依次为沪市 A 股、深市 A 股、北交所股票、港股、美股、沪市 ETF、深市 ETF、
// 沪市指数、深市指数、北交所指数。所有规则均未命中时不补后缀、不返回错误，原样返回清理后的纯代码。
func StockPrefix(code string) string {
	code = CleanCode(code)
	// 分支顺序即匹配优先级：先判定 A 股与海外股票，再判定 ETF，最后判定指数，避免前缀规则互相覆盖。
	switch {
	case isSHStock(code):
		return fmt.Sprintf("%s.%s", code, MarketSH.String())
	case isSZStock(code):
		return fmt.Sprintf("%s.%s", code, MarketSZ.String())
	case isBJStock(code):
		return fmt.Sprintf("%s.%s", code, MarketBJ.String())
	case isHKStock(code):
		return fmt.Sprintf("%s.%s", code, MarketHK.String())
	case isUSAStock(code):
		return fmt.Sprintf("%s.%s", code, MarketUSA.String())
	case isSHETF(code):
		return fmt.Sprintf("%s.%s", code, MarketSH.String())
	case isSZETF(code):
		return fmt.Sprintf("%s.%s", code, MarketSZ.String())
	case isSHIndex(code):
		return fmt.Sprintf("%s.%s", code, MarketSH.String())
	case isSZIndex(code):
		return fmt.Sprintf("%s.%s", code, MarketSZ.String())
	case isBJIndex(code):
		return fmt.Sprintf("%s.%s", code, MarketBJ.String())
	}
	// 所有规则均未命中时保持原纯代码返回，不强行补后缀。
	return code
}

// IndexPrefix 根据纯代码规则为指数代码补全交易所后缀，仅覆盖沪深北三地指数；无法识别时原样返回。
//
// 参数 `code` 为指数纯代码（允许携带旧后缀，先经 `CleanCode` 清理）；命中沪市、深市、北交所指数前缀
// 规则时返回 `代码.对应后缀`，其余标的不补后缀、不返回错误，原样返回清理后的纯代码。与 `StockPrefix`
// 的区别在于本函数不处理股票、ETF 与海外市场代码。
func IndexPrefix(code string) string {
	code = CleanCode(code)
	switch {
	case isSHIndex(code):
		return fmt.Sprintf("%s.%s", code, MarketSH.String())
	case isSZIndex(code):
		return fmt.Sprintf("%s.%s", code, MarketSZ.String())
	case isBJIndex(code):
		return fmt.Sprintf("%s.%s", code, MarketBJ.String())
	}
	return code
}

// IsStock 判断带后缀的证券代码是否为沪深北 A 股股票，不含港股、美股、ETF 与指数。
//
// 参数 `symbol` 为带两位后缀的代码；内部分别调用 `IsSHStock`、`IsSZStock`、`IsBJStock`，
// 三者任一命中返回 `true`，否则返回 `false`。函数不返回错误，格式不符的输入按未命中处理。
func IsStock(symbol string) bool {
	return IsSZStock(symbol) || IsSHStock(symbol) || IsBJStock(symbol)
}

// IsSZStock 判断带后缀的代码是否为深圳 A 股代码。
//
// 参数 `symbol` 为 `六位代码.SZ` 形式的深市代码，首尾空白与大小写不敏感。判定须同时满足总长度为
// 九位（六位代码、点号、两位后缀）、后缀为 `SZ`、代码段命中深市前缀规则（0 开头或 30 开头创业板），
// 三项任一不满足返回 `false`；函数不返回错误。
func IsSZStock(symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	market := symbol[len(symbol)-2:]
	code := symbol[:len(symbol)-3]
	// 长度须为九位（六位代码、点号、两位后缀），先排除格式不符的输入，避免后续前缀判定失真。
	return len(symbol) == 9 && market == MarketSZ.String() && isSZStock(code)
}

// IsSHStock 判断带后缀的代码是否为上海 A 股代码。
//
// 参数 `symbol` 为 `六位代码.SH` 形式的沪市代码，首尾空白与大小写不敏感。判定须同时满足总长度为
// 九位、后缀为 `SH`、代码段命中沪市前缀规则（6 开头），三项任一不满足返回 `false`；函数不返回错误。
func IsSHStock(symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	market := symbol[len(symbol)-2:]
	code := symbol[:len(symbol)-3]
	// 长度须为九位（六位代码、点号、两位后缀），先排除格式不符的输入，避免后续前缀判定失真。
	return len(symbol) == 9 && market == MarketSH.String() && isSHStock(code)
}

// IsBJStock 判断带后缀的代码是否为北交所股票代码。
//
// 参数 `symbol` 为 `六位代码.BJ` 形式的北交所代码，首尾空白与大小写不敏感。判定须同时满足总长度为
// 九位、后缀为 `BJ`、代码段命中北交所前缀规则（92 开头），三项任一不满足返回 `false`；函数不返回错误。
func IsBJStock(symbol string) bool {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	market := symbol[len(symbol)-2:]
	code := symbol[:len(symbol)-3]
	// 长度须为九位（六位代码、点号、两位后缀），先排除格式不符的输入，避免后续前缀判定失真。
	return len(symbol) == 9 && market == MarketBJ.String() && isBJStock(code)
}

// IsETF 判断带后缀的代码是否为沪深 ETF。
//
// 参数 `symbol` 为 `六位代码.SH` 或 `六位代码.SZ` 形式的基金代码；先做九位长度短路校验，长度不符
// 直接返回 `false`，再归一化大小写并取出后缀与代码段。代码段统一转小写后匹配对应市场 ETF 前缀规则，
// 不匹配沪市或深市 ETF 规则时返回 `false`；函数不返回错误。
func IsETF(symbol string) bool {
	// 长度不符直接短路返回，避免后续按固定位置切片造成越界。
	if len(symbol) != 9 {
		return false
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	market := symbol[len(symbol)-2:]
	code := symbol[:len(symbol)-3]

	// ETF 前缀规则不区分大小写，统一转小写后再匹配。
	code = strings.ToLower(code)
	switch {
	case market == MarketSH.String() && isSHETF(code):
		return true
	case market == MarketSZ.String() && isSZETF(code):
		return true
	}
	return false
}

// IsIndex 判断带后缀的代码是否为沪深北指数。
//
// 参数 `symbol` 为带两位后缀的指数代码；先做九位长度短路校验，长度不符返回 `false`，再按后缀分别
// 匹配沪市（000 开头及特殊代码 999999）、深市（399 开头）、北交所（899 开头）指数前缀规则，
// 任一市场命中返回 `true`，否则返回 `false`；函数不返回错误。
func IsIndex(symbol string) bool {
	// 长度不符直接短路返回，避免后续按固定位置切片造成越界。
	if len(symbol) != 9 {
		return false
	}
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	market := symbol[len(symbol)-2:]
	code := symbol[:len(symbol)-3]
	switch {
	case market == MarketSH.String() && isSHIndex(code):
		return true
	case market == MarketSZ.String() && isSZIndex(code):
		return true
	case market == MarketBJ.String() && isBJIndex(code):
		return true
	}
	return false
}

// isSHStock 判断代码是否符合沪市 A 股前缀规则；参数 `code` 为待判定代码，长度非六位返回 `false`。
func isSHStock(code string) bool {
	code = CleanCode(code)
	if len(code) != 6 {
		return false
	}
	// 沪市 A 股：6 开头（600、601、603、605 等）。
	return code[:1] == "6"
}

// isSZStock 判断代码是否符合深市 A 股前缀规则；参数 `code` 为待判定代码，长度非六位返回 `false`。
func isSZStock(code string) bool {
	code = CleanCode(code)
	if len(code) != 6 {
		return false
	}
	// 深市 A 股：0 开头（000、001、002、003），或 30 开头的创业板（300、301）。
	return code[:1] == "0" || code[:2] == "30"
}

// isBJStock 判断代码是否符合北交所股票前缀规则；参数 `code` 为待判定代码，长度非六位返回 `false`。
func isBJStock(code string) bool {
	code = CleanCode(code)
	if len(code) != 6 {
		return false
	}
	// 北交所股票：92 开头。
	return code[:2] == "92"
}

// isHKStock 判断代码是否符合港股代码规则；参数 `code` 为待判定代码，非五位纯数字返回 `false`。
func isHKStock(code string) bool {
	code = CleanCode(code)
	if len(code) != 5 {
		return false
	}
	// 港股代码为五位纯数字。
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// isUSAStock 判断代码是否符合美股代码规则；参数 `code` 为待判定代码，非一至五位字母返回 `false`。
func isUSAStock(code string) bool {
	code = CleanCode(code)
	// 美股代码为一至五位字母，按正则整体匹配。
	matched, _ := regexp.MatchString(`^[A-Za-z]{1,5}$`, code)
	return matched
}

// CleanCode 剥离证券代码中的市场后缀，返回点号之前的纯代码段，不校验代码合法性。
//
// 参数 `code` 为原始代码串；空串原样返回。存在点号时只保留第一个点号之前的部分，点号之后的后缀一律
// 丢弃；不含点号时视为纯代码原样返回。返回值始终为字符串、不返回错误，代码段是否合法由调用方另行判断。
func CleanCode(code string) string {
	if code == "" {
		return ""
	}
	// 仅按第一个点号切分，避免代码段本身含点号时误截断；后缀整体丢弃。
	if idx := strings.Index(code, "."); idx != -1 {
		return code[:idx]
	}
	return code
}

// isSHETF 判断已清理后缀的代码段是否符合沪市 ETF 前缀规则；参数 `code` 长度非六位返回 `false`。
func isSHETF(code string) bool {
	if len(code) != 6 {
		return false
	}
	// 沪市 ETF：50/51/52/53/56/58 开头。
	switch code[:2] {
	case "50", "51", "52", "53", "56", "58":
		return true
	}

	return false
}

// isSZETF 判断已清理后缀的代码段是否符合深市 ETF 前缀规则；参数 `code` 长度非六位返回 `false`。
func isSZETF(code string) bool {
	if len(code) != 6 {
		return false
	}
	// 深市 ETF：15/16 开头。
	return code[:2] == "15" || code[:2] == "16"
}

// isSHIndex 判断已清理后缀的代码段是否符合沪市指数前缀规则；参数 `code` 长度非六位返回 `false`。
func isSHIndex(code string) bool {
	if len(code) != 6 {
		return false
	}
	// 沪市指数：000 开头，另有特殊代码 999999。
	return code[:3] == "000" || code == "999999"
}

// isSZIndex 判断已清理后缀的代码段是否符合深市指数前缀规则；参数 `code` 长度非六位返回 `false`。
func isSZIndex(code string) bool {
	if len(code) != 6 {
		return false
	}
	// 深市指数：399 开头。
	return code[:3] == "399"
}

// isBJIndex 判断已清理后缀的代码段是否符合北交所指数前缀规则；参数 `code` 长度非六位返回 `false`。
func isBJIndex(code string) bool {
	if len(code) != 6 {
		return false
	}
	// 北交所指数：899 开头。
	return code[:3] == "899"
}

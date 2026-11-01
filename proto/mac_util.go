package proto

import (
	"fmt"
	"strconv"
	"strings"
)

// makeMACCode8 把证券代码复制到八字节定长数组，长度不足时尾部以零填充。
//
// 参数 `code` 为证券代码字符串；返回八字节定长数组。
func makeMACCode8(code string) [8]byte {
	var out [8]byte
	copy(out[:], code)
	return out
}

// makeMACCode21 把证券代码复制到二十一字节定长数组，长度不足时尾部以零填充。
//
// 参数 `code` 为证券代码字符串；返回二十一字节定长数组。
func makeMACCode21(code string) [21]byte {
	var out [21]byte
	copy(out[:], code)
	return out
}

// makeMACCode22 把证券代码复制到二十二字节定长数组，长度不足时尾部以零填充。
//
// 参数 `code` 为证券代码字符串；返回二十二字节定长数组。
func makeMACCode22(code string) [22]byte {
	var out [22]byte
	copy(out[:], code)
	return out
}

// ExchangeMACBoardCode 把板块符号按前缀规则映射为 MAC 行情协议使用的板块数字编码。
//
// 参数 `boardSymbol` 为带市场前缀或纯数字的板块符号；返回映射后的板块数字编码，
// 前缀去除后的数字或纯数字无法解析时返回错误。
// 美股、港股前缀分别落入三万与两万区段，A 股指数与行业板块按各自基准码平移；
// 无前缀的纯数字编码原样透传。
func ExchangeMACBoardCode(boardSymbol string) (uint32, error) {
	switch {
	case strings.HasPrefix(boardSymbol, "US"):
		// 美股板块去掉前缀后加入三万区段基准。
		value, err := strconv.Atoi(strings.TrimPrefix(boardSymbol, "US"))
		if err != nil {
			return 0, err
		}
		return uint32(30000 + value), nil
	case strings.HasPrefix(boardSymbol, "HK"):
		// 港股板块去掉前缀后加入两万区段基准。
		value, err := strconv.Atoi(strings.TrimPrefix(boardSymbol, "HK"))
		if err != nil {
			return 0, err
		}
		return uint32(20000 + value), nil
	case strings.HasPrefix(boardSymbol, "000"):
		// 零开头板块加入三万一千区段基准。
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(31000 + value), nil
	case strings.HasPrefix(boardSymbol, "399"):
		// 三九九开头指数板块相对基准平移到三万区段。
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(value - 399000 + 30000), nil
	case strings.HasPrefix(boardSymbol, "899"):
		// 八九九开头板块相对基准平移到三万二千区段。
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(value - 899000 + 32000), nil
	case strings.HasPrefix(boardSymbol, "88"):
		// 八八开头板块相对基准平移到两万区段。
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(value - 880000 + 20000), nil
	default:
		// 无前缀的纯数字板块编码原样透传；非数字则返回解析错误。
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, fmt.Errorf("invalid board symbol %q: %w", boardSymbol, err)
		}
		return uint32(value), nil
	}
}

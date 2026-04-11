package proto

import (
	"fmt"
	"strconv"
	"strings"
)

func makeMACCode8(code string) [8]byte {
	var out [8]byte
	copy(out[:], code)
	return out
}

func makeMACCode21(code string) [21]byte {
	var out [21]byte
	copy(out[:], code)
	return out
}

func makeMACCode22(code string) [22]byte {
	var out [22]byte
	copy(out[:], code)
	return out
}

func ExchangeMACBoardCode(boardSymbol string) (uint32, error) {
	switch {
	case strings.HasPrefix(boardSymbol, "US"):
		value, err := strconv.Atoi(strings.TrimPrefix(boardSymbol, "US"))
		if err != nil {
			return 0, err
		}
		return uint32(30000 + value), nil
	case strings.HasPrefix(boardSymbol, "HK"):
		value, err := strconv.Atoi(strings.TrimPrefix(boardSymbol, "HK"))
		if err != nil {
			return 0, err
		}
		return uint32(20000 + value), nil
	case strings.HasPrefix(boardSymbol, "000"):
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(31000 + value), nil
	case strings.HasPrefix(boardSymbol, "399"):
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(value - 399000 + 30000), nil
	case strings.HasPrefix(boardSymbol, "899"):
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(value - 899000 + 32000), nil
	case strings.HasPrefix(boardSymbol, "88"):
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, err
		}
		return uint32(value - 880000 + 20000), nil
	default:
		value, err := strconv.Atoi(boardSymbol)
		if err != nil {
			return 0, fmt.Errorf("invalid board symbol %q: %w", boardSymbol, err)
		}
		return uint32(value), nil
	}
}

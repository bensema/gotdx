package gotdx

import (
	"errors"

	"github.com/bensema/gotdx/proto"
)

var ErrMarketCodeCount = errors.New("market code count error")

func executeProtocol[T any](client *Client, protocol proto.Protocol[T]) (T, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	return executeProtocolLocked(client, protocol)
}

func executeProtocolLocked[T any](client *Client, protocol proto.Protocol[T]) (T, error) {
	var zero T

	header, payload, err := client.exchange(protocol)
	if err != nil {
		return zero, err
	}
	if err := protocol.ParseResponse(header, payload); err != nil {
		return zero, err
	}
	return protocol.Response(), nil
}

func makeCode6(code string) [6]byte {
	var out [6]byte
	copy(out[:], code)
	return out
}

func makeCode9(code string) [9]byte {
	var out [9]byte
	copy(out[:], code)
	return out
}

func makeCode23(code string) [23]byte {
	var out [23]byte
	copy(out[:], code)
	return out
}

func makeCode22(code string) [22]byte {
	var out [22]byte
	copy(out[:], code)
	return out
}

func makeFixed40(value string) [40]byte {
	var out [40]byte
	copy(out[:], value)
	return out
}

func makeFixed80(value string) [80]byte {
	var out [80]byte
	copy(out[:], value)
	return out
}

func makeFixed300(value string) [300]byte {
	var out [300]byte
	copy(out[:], value)
	return out
}

func makeFixed43(value string) [43]byte {
	var out [43]byte
	copy(out[:], value)
	return out
}

func makeStocks(markets []uint8, codes []string) ([]proto.Stock, error) {
	if len(markets) != len(codes) {
		return nil, ErrMarketCodeCount
	}

	stocks := make([]proto.Stock, 0, len(markets))
	for i, market := range markets {
		stocks = append(stocks, proto.Stock{
			Market: market,
			Code:   codes[i],
		})
	}
	return stocks, nil
}

func makeExStocks(categories []uint8, codes []string) ([]proto.ExStock, error) {
	if len(categories) != len(codes) {
		return nil, ErrMarketCodeCount
	}

	stocks := make([]proto.ExStock, 0, len(categories))
	for i, category := range categories {
		stocks = append(stocks, proto.ExStock{
			Category: category,
			Code:     codes[i],
		})
	}
	return stocks, nil
}

func quotesSortReverse(sortType uint16, reverse bool) uint16 {
	if sortType == SortCode {
		return 0
	}
	if reverse {
		return 2
	}
	return 1
}

package gotdx

import (
	"testing"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

func TestNewKeepsMainAndExAddresses(t *testing.T) {
	client := New(
		WithTCPAddress("1.1.1.1:7709"),
		WithExTCPAddress("2.2.2.2:7727"),
	)

	if client.mode != clientModeMain {
		t.Fatalf("unexpected client mode: %d", client.mode)
	}
	if client.opt.TCPAddress != "1.1.1.1:7709" {
		t.Fatalf("unexpected main tcp address: %q", client.opt.TCPAddress)
	}
	if client.opt.ExTCPAddress != "2.2.2.2:7727" {
		t.Fatalf("unexpected ex tcp address: %q", client.opt.ExTCPAddress)
	}
}

func TestNewExUsesExAddressAsPrimary(t *testing.T) {
	client := NewEx(
		WithTCPAddress("1.1.1.1:7709"),
		WithExTCPAddress("2.2.2.2:7727"),
	)

	if client.mode != clientModeEx {
		t.Fatalf("unexpected client mode: %d", client.mode)
	}
	if client.opt.TCPAddress != "2.2.2.2:7727" {
		t.Fatalf("unexpected primary tcp address: %q", client.opt.TCPAddress)
	}
	if client.opt.ExTCPAddress != "2.2.2.2:7727" {
		t.Fatalf("unexpected ex tcp address: %q", client.opt.ExTCPAddress)
	}
}

func TestNewMACUsesMacAddressAsPrimary(t *testing.T) {
	client := NewMAC(
		WithMacTCPAddress("3.3.3.3:7709"),
		WithMacExTCPAddress("4.4.4.4:7727"),
	)

	if client.mode != clientModeMacMain {
		t.Fatalf("unexpected client mode: %d", client.mode)
	}
	if client.opt.TCPAddress != "3.3.3.3:7709" {
		t.Fatalf("unexpected primary tcp address: %q", client.opt.TCPAddress)
	}
	if client.opt.MacTCPAddress != "3.3.3.3:7709" {
		t.Fatalf("unexpected mac tcp address: %q", client.opt.MacTCPAddress)
	}
}

func TestNewMACExUsesMacExAddressAsPrimary(t *testing.T) {
	client := NewMACEx(
		WithMacTCPAddress("3.3.3.3:7709"),
		WithMacExTCPAddress("4.4.4.4:7727"),
	)

	if client.mode != clientModeMacEx {
		t.Fatalf("unexpected client mode: %d", client.mode)
	}
	if client.opt.TCPAddress != "4.4.4.4:7727" {
		t.Fatalf("unexpected primary tcp address: %q", client.opt.TCPAddress)
	}
	if client.opt.MacExTCPAddress != "4.4.4.4:7727" {
		t.Fatalf("unexpected mac ex tcp address: %q", client.opt.MacExTCPAddress)
	}
}

func TestApplyTurnoverHelpers(t *testing.T) {
	shares := map[stockKey]float64{
		{Market: types.MarketSZ.Uint8(), Code: "000001"}: 100000,
	}

	quotes := []proto.SecurityQuote{{
		Market: types.MarketSZ.Uint8(),
		Code:   "000001",
		Vol:    100000,
	}}
	applyTurnoverToSecurityQuotes(quotes, shares)
	if quotes[0].Turnover != 1.0 {
		t.Fatalf("unexpected security quote turnover: %.2f", quotes[0].Turnover)
	}

	quoteList := []proto.QuoteListItem{{
		Market: types.MarketSZ.Uint8(),
		Code:   "000001",
		Vol:    150000,
	}}
	applyTurnoverToQuoteList(quoteList, shares)
	if quoteList[0].Turnover != 1.5 {
		t.Fatalf("unexpected quote list turnover: %.2f", quoteList[0].Turnover)
	}

	bars := []proto.SecurityBar{{Vol: 2000000}}
	applyTurnoverToBars(bars, 2000)
	if bars[0].Turnover != 10.0 {
		t.Fatalf("unexpected bar turnover: %.2f", bars[0].Turnover)
	}

	reply := &proto.GetVolumeProfileReply{
		Market: types.MarketSZ.Uint8(),
		Code:   "000001",
		Vol:    200000,
	}
	applyTurnoverToVolumeProfile(reply, 100000)
	if reply.Turnover != 2.0 {
		t.Fatalf("unexpected volume profile turnover: %.2f", reply.Turnover)
	}
}

func TestApplyDecimalPointToQuotes(t *testing.T) {
	decimals := map[stockKey]int8{{Market: types.MarketSZ.Uint8(), Code: "000001"}: 3}
	quotes := []proto.SecurityQuote{{
		Market:    types.MarketSZ.Uint8(),
		Code:      "000001",
		Close:     12.34,
		Price:     12.34,
		PreClose:  12.30,
		LastClose: 12.30,
		Open:      12.31,
		High:      12.50,
		Low:       12.20,
		NegPrice:  12.10,
		BidLevels: []proto.Level{{Price: 12.33}},
		AskLevels: []proto.Level{{Price: 12.35}},
		Bid1:      12.33,
		Ask1:      12.35,
	}}
	applyDecimalPointToSecurityQuotes(quotes, decimals)
	if quotes[0].Close != 1.234 || quotes[0].BidLevels[0].Price != 1.233 || quotes[0].Ask1 != 1.235 {
		t.Fatalf("unexpected adjusted security quote: %+v", quotes[0])
	}

	zeroDecimal := []proto.QuoteListItem{{Market: types.MarketSH.Uint8(), Code: "000001", Close: 12.34}}
	applyDecimalPointToQuoteList(zeroDecimal, map[stockKey]int8{{Market: types.MarketSH.Uint8(), Code: "000001"}: 0})
	if zeroDecimal[0].Close != 1234 {
		t.Fatalf("unexpected zero decimal adjustment: %+v", zeroDecimal[0])
	}

	items := []proto.QuoteListItem{{
		Market:    types.MarketSZ.Uint8(),
		Code:      "000001",
		Close:     12.34,
		PreClose:  12.30,
		Open:      12.31,
		High:      12.50,
		Low:       12.20,
		NegPrice:  12.10,
		BidLevels: []proto.Level{{Price: 12.33}},
		AskLevels: []proto.Level{{Price: 12.35}},
	}}
	applyDecimalPointToQuoteList(items, decimals)
	if items[0].Close != 1.234 || items[0].BidLevels[0].Price != 1.233 {
		t.Fatalf("unexpected adjusted quote list item: %+v", items[0])
	}
}

func TestApplyMACSymbolBarTurnover(t *testing.T) {
	items := []proto.MACSymbolBar{{Vol: 500000, FloatShares: 1000}}
	applyMACSymbolBarTurnover(items)
	if items[0].Turnover != 5 {
		t.Fatalf("unexpected mac symbol bar turnover: %.2f", items[0].Turnover)
	}
}

func TestRound2(t *testing.T) {
	if got := round2(1.235); got != 1.24 {
		t.Fatalf("unexpected rounded value: %.2f", got)
	}
}

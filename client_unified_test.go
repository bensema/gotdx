package gotdx

import (
	"testing"

	"github.com/bensema/gotdx/proto"
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
		{Market: MarketSZ, Code: "000001"}: 1000000,
	}

	quotes := []proto.SecurityQuote{{
		Market: MarketSZ,
		Code:   "000001",
		Vol:    12345,
	}}
	applyTurnoverToSecurityQuotes(quotes, shares)
	if quotes[0].Turnover != 123.45 {
		t.Fatalf("unexpected security quote turnover: %.2f", quotes[0].Turnover)
	}

	quoteList := []proto.QuoteListItem{{
		Market: MarketSZ,
		Code:   "000001",
		Vol:    23456,
	}}
	applyTurnoverToQuoteList(quoteList, shares)
	if quoteList[0].Turnover != 234.56 {
		t.Fatalf("unexpected quote list turnover: %.2f", quoteList[0].Turnover)
	}

	bars := []proto.SecurityBar{{Vol: 12345}}
	applyTurnoverToBars(bars, 1000000)
	if bars[0].Turnover != 1.23 {
		t.Fatalf("unexpected bar turnover: %.2f", bars[0].Turnover)
	}

	reply := &proto.GetVolumeProfileReply{
		Market: MarketSZ,
		Code:   "000001",
		Vol:    34567,
	}
	applyTurnoverToVolumeProfile(reply, 1000000)
	if reply.Turnover != 345.67 {
		t.Fatalf("unexpected volume profile turnover: %.2f", reply.Turnover)
	}
}

func TestRound2(t *testing.T) {
	if got := round2(1.235); got != 1.24 {
		t.Fatalf("unexpected rounded value: %.2f", got)
	}
}

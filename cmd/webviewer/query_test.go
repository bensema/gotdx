package main

import (
	"testing"

	"github.com/bensema/gotdx/proto"
)

func TestParseExTableRows(t *testing.T) {
	rows := parseExTableRows("42#IMCI|上期有色,42#T001|通达信商品,")
	if len(rows) != 2 {
		t.Fatalf("unexpected rows len: %d", len(rows))
	}
	if rows[0][1] != "42" || rows[0][2] != "IMCI" || rows[0][3] != "上期有色" {
		t.Fatalf("unexpected first row: %#v", rows[0])
	}
}

func TestParseExTableDetailRows(t *testing.T) {
	columns, rows := parseExTableDetailRows("42#IMCI|3|2,42#T001|4|5|6")
	if len(columns) != 4 {
		t.Fatalf("unexpected columns len: %d", len(columns))
	}
	if len(rows) != 2 || rows[1][3] != "6" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
}

func TestExpandUint8List(t *testing.T) {
	values, err := expandUint8List([]uint8{74}, 3, "categories")
	if err != nil {
		t.Fatalf("expand failed: %v", err)
	}
	if len(values) != 3 || values[0] != 74 || values[1] != 74 || values[2] != 74 {
		t.Fatalf("unexpected expanded values: %#v", values)
	}
}

func TestMethodDefsStockFirst(t *testing.T) {
	if len(methodDefs) == 0 {
		t.Fatal("expected method defs")
	}
	for i := 0; i < len(methodDefs); i++ {
		if methodDefs[i].Group == "连接状态" {
			for j := i + 1; j < len(methodDefs); j++ {
				if methodDefs[j].Group == "股票快照" || methodDefs[j].Group == "股票分时" || methodDefs[j].Group == "股票指数" || methodDefs[j].Group == "股票监控" || methodDefs[j].Group == "股票资料" {
					t.Fatalf("stock group %q appears after connection group", methodDefs[j].Group)
				}
			}
			break
		}
	}
	if methodDefs[0].Group != "股票快照" {
		t.Fatalf("expected stock methods first, got %q", methodDefs[0].Group)
	}
}

func TestRowsFromUnusualIncludesUnusualType(t *testing.T) {
	rows := rowsFromUnusual([]proto.UnusualData{{
		Index:       1,
		Market:      0,
		Code:        "000001",
		Time:        "09:30:00",
		Desc:        "加速拉升",
		Value:       "1.23%",
		UnusualType: 4,
	}})
	if len(rows) != 1 {
		t.Fatalf("unexpected rows len: %d", len(rows))
	}
	if len(rows[0]) != 7 || rows[0][6] != "4" {
		t.Fatalf("unexpected unusual row: %#v", rows[0])
	}
}

func TestRowsIncludeTurnoverColumns(t *testing.T) {
	detailRows := rowsFromQuoteDetail([]proto.SecurityQuote{{
		Market:     0,
		Code:       "000001",
		ServerTime: "09:30:00",
		Price:      10.01,
		Open:       9.80,
		High:       10.20,
		Low:        9.70,
		Vol:        100,
		Amount:     1000,
		Turnover:   1.23,
	}})
	if len(detailRows) != 1 || len(detailRows[0]) != 10 || detailRows[0][9] != "1.23" {
		t.Fatalf("unexpected quote detail rows: %#v", detailRows)
	}

	listRows := rowsFromQuoteList([]proto.QuoteListItem{{
		Market:    0,
		Code:      "000001",
		Price:     10.01,
		PreClose:  9.91,
		Vol:       100,
		Amount:    1000,
		RiseSpeed: 0.56,
		Turnover:  2.34,
	}})
	if len(listRows) != 1 || len(listRows[0]) != 9 || listRows[0][8] != "2.34" {
		t.Fatalf("unexpected quote list rows: %#v", listRows)
	}

	barRows := rowsFromSecurityBars([]proto.SecurityBar{{
		DateTime: "2026-04-12 15:00:00",
		Open:     10,
		High:     11,
		Low:      9,
		Close:    10.5,
		Vol:      12345,
		Amount:   45678,
		Turnover: 3.45,
	}})
	if len(barRows) != 1 || len(barRows[0]) != 8 || barRows[0][7] != "3.45" {
		t.Fatalf("unexpected bar rows: %#v", barRows)
	}

	profileRows := rowsFromVolumeProfile([]proto.VolumeProfileItem{{
		Price: 10.01,
		Vol:   100,
		Buy:   60,
		Sell:  40,
	}}, 4.56)
	if len(profileRows) != 1 || len(profileRows[0]) != 5 || profileRows[0][4] != "4.56" {
		t.Fatalf("unexpected volume profile rows: %#v", profileRows)
	}
}

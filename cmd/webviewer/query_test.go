package main

import (
	"strings"
	"testing"

	"github.com/bensema/gotdx/proto"
)

func hasMethodDef(key string) bool {
	for _, def := range methodDefs {
		if def.Key == key {
			return true
		}
	}
	return false
}

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

func TestMethodDefsIncludeNewComparisonMethods(t *testing.T) {
	keys := []string{
		"stock_file_meta",
		"stock_file_download",
		"stock_file_full",
		"stock_table_file",
		"stock_csv_file",
		"stock_block_flat",
		"ex_file_meta",
		"ex_file_download",
		"mac_board_count",
		"mac_board_members_quotes_dynamic",
		"mac_quotes",
		"mac_ex_board_count",
		"mac_ex_quotes",
	}
	for _, key := range keys {
		if !hasMethodDef(key) {
			t.Fatalf("missing method def %q", key)
		}
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
	if len(barRows) != 1 || len(barRows[0]) != 11 || barRows[0][8] != "3.45" {
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

func TestRowsFromFileMeta(t *testing.T) {
	item := &proto.GetFileMetaReply{
		Size:     1234,
		Unknown1: 7,
		Unknown2: 9,
	}
	copy(item.HashValue[:], []byte("hash-value-demo"))

	rows := rowsFromFileMeta(item)
	if len(rows) != 4 {
		t.Fatalf("unexpected rows len: %d", len(rows))
	}
	if rows[0][1] != "1234" || rows[1][1] != "7" || rows[3][1] != "9" {
		t.Fatalf("unexpected file meta rows: %#v", rows)
	}
	if !strings.HasPrefix(rows[2][1], "68617368") {
		t.Fatalf("unexpected hash row: %#v", rows[2])
	}
}

func TestRowsFromMACQuoteChart(t *testing.T) {
	rows := rowsFromMACQuoteChart([]proto.MACQuoteChartItem{{
		Time:     "09:30:00",
		Price:    10.1,
		Avg:      10.0,
		Vol:      1234,
		Momentum: 0.5,
	}})
	if len(rows) != 1 {
		t.Fatalf("unexpected rows len: %d", len(rows))
	}
	if len(rows[0]) != 5 || rows[0][0] != "09:30:00" || rows[0][3] != "1234" || rows[0][4] != "0.50" {
		t.Fatalf("unexpected mac quote chart rows: %#v", rows)
	}
}

func TestParseMACBoardMembersQuotesFieldBitmap(t *testing.T) {
	defaultBitmap, err := parseMACBoardMembersQuotesFieldBitmap("")
	if err != nil {
		t.Fatalf("parse default bitmap failed: %v", err)
	}
	if defaultBitmap[0] != 0xff || defaultBitmap[1] != 0xfc {
		t.Fatalf("unexpected default bitmap: %#v", defaultBitmap)
	}

	fullBitmap, err := parseMACBoardMembersQuotesFieldBitmap("full")
	if err != nil {
		t.Fatalf("parse full bitmap failed: %v", err)
	}
	if fullBitmap[0] != 0xff || fullBitmap[19] != 0xff {
		t.Fatalf("unexpected full bitmap: %#v", fullBitmap)
	}

	customBitmap, err := parseMACBoardMembersQuotesFieldBitmap("3100000000000000000000000000000000000000")
	if err != nil {
		t.Fatalf("parse custom bitmap failed: %v", err)
	}
	if customBitmap[0] != 0x31 || customBitmap[1] != 0x00 {
		t.Fatalf("unexpected custom bitmap: %#v", customBitmap)
	}
}

func TestRowsFromMACBoardMemberQuotesDynamic(t *testing.T) {
	reply := &proto.MACBoardMembersQuotesDynamicReply{
		ActiveFields: []proto.MACDynamicFieldDef{
			{Name: "pre_close"},
			{Name: "close"},
			{Name: "vol"},
		},
		Stocks: []proto.MACBoardMemberQuoteDynamicItem{{
			Market: 1,
			Symbol: "600000",
			Name:   "BANK",
			Values: map[string]any{
				"pre_close": 10.1,
				"close":     10.5,
				"vol":       uint32(1234),
			},
		}},
	}

	columns := columnsFromMACBoardMemberQuotesDynamic(reply)
	if len(columns) != 6 || columns[3] != "pre_close" || columns[5] != "vol" {
		t.Fatalf("unexpected dynamic columns: %#v", columns)
	}

	rows := rowsFromMACBoardMemberQuotesDynamic(reply)
	if len(rows) != 1 || len(rows[0]) != 6 {
		t.Fatalf("unexpected dynamic rows: %#v", rows)
	}
	if rows[0][0] != "1" || rows[0][1] != "600000" || rows[0][3] != "10.10" || rows[0][5] != "1234" {
		t.Fatalf("unexpected dynamic row values: %#v", rows[0])
	}
}

func TestRowsFromMACDynamicFieldDefs(t *testing.T) {
	rows := rowsFromMACDynamicFieldDefs([]proto.MACDynamicFieldDef{{
		Bit:         0x25,
		Name:        "speed_pct",
		Format:      "float32",
		Description: "涨速",
	}})
	if len(rows) != 1 || len(rows[0]) != 4 {
		t.Fatalf("unexpected field rows: %#v", rows)
	}
	if rows[0][0] != "37" || rows[0][1] != "speed_pct" || rows[0][3] != "涨速" {
		t.Fatalf("unexpected field row values: %#v", rows[0])
	}
}

func TestNormalizeTableRows(t *testing.T) {
	columns, rows := normalizeTableRows([][]string{
		{"a", "b"},
		{"1"},
	}, "col")
	if len(columns) != 2 || columns[0] != "col_0" || columns[1] != "col_1" {
		t.Fatalf("unexpected columns: %#v", columns)
	}
	if len(rows) != 2 || len(rows[1]) != 2 || rows[1][0] != "1" || rows[1][1] != "" {
		t.Fatalf("unexpected normalized rows: %#v", rows)
	}
}

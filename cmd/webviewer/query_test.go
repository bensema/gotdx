package main

import "testing"

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

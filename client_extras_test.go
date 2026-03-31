package gotdx

import "testing"

func TestParsePipeTableContent(t *testing.T) {
	rows := parsePipeTableContent([]byte("a|b|c\n1|2|3\n\n"))
	if len(rows) != 2 {
		t.Fatalf("unexpected row count: %d", len(rows))
	}
	if rows[0][0] != "a" || rows[1][2] != "3" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

func TestParseCSVContent(t *testing.T) {
	rows, err := parseCSVContent([]byte("a,b,c\n1,2,3\n"))
	if err != nil {
		t.Fatalf("parseCSVContent failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("unexpected row count: %d", len(rows))
	}
	if rows[0][1] != "b" || rows[1][2] != "3" {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}

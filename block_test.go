package gotdx

import "testing"

func buildBlockData() []byte {
	data := make([]byte, 384+2+2800)
	data[384] = 1
	copy(data[386:395], []byte{'T', 'E', 'S', 'T'})
	data[395] = 2
	data[397] = 3
	copy(data[399:406], []byte{'6', '0', '0', '0', '0', '0'})
	copy(data[406:413], []byte{'0', '0', '0', '0', '0', '1'})
	return data
}

func TestParseBlockFlat(t *testing.T) {
	items, err := ParseBlockFlat(buildBlockData())
	if err != nil {
		t.Fatalf("ParseBlockFlat failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].BlockName != "TEST" || items[0].Code != "600000" || items[1].Code != "000001" {
		t.Fatalf("unexpected block items: %+v", items)
	}
}

func TestParseBlockGroups(t *testing.T) {
	items, err := ParseBlockGroups(buildBlockData())
	if err != nil {
		t.Fatalf("ParseBlockGroups failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected group count: %d", len(items))
	}
	if items[0].BlockName != "TEST" || items[0].StockCount != 2 || len(items[0].Codes) != 2 {
		t.Fatalf("unexpected block group: %+v", items[0])
	}
}

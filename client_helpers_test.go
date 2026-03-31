package gotdx

import "testing"

func TestMakeStocks(t *testing.T) {
	stocks, err := makeStocks([]uint8{MarketSZ, MarketSH}, []string{"000001", "600000"})
	if err != nil {
		t.Fatalf("makeStocks failed: %v", err)
	}
	if len(stocks) != 2 {
		t.Fatalf("unexpected stock len: %d", len(stocks))
	}
	if stocks[0].Market != MarketSZ || stocks[0].Code != "000001" {
		t.Fatalf("unexpected first stock: %+v", stocks[0])
	}
	if stocks[1].Market != MarketSH || stocks[1].Code != "600000" {
		t.Fatalf("unexpected second stock: %+v", stocks[1])
	}
}

func TestMakeStocksCountMismatch(t *testing.T) {
	if _, err := makeStocks([]uint8{MarketSZ}, []string{"000001", "600000"}); err == nil {
		t.Fatal("expected error on count mismatch")
	}
}

func TestMakeFixedBuffers(t *testing.T) {
	code := makeCode6("600000EXTRA")
	if string(code[:]) != "600000" {
		t.Fatalf("unexpected code buffer: %q", string(code[:]))
	}

	code9 := makeCode9("TSLA-EXTRA")
	if string(code9[:4]) != "TSLA" {
		t.Fatalf("unexpected code9 buffer: %q", string(code9[:4]))
	}

	code22 := makeCode22("TSLA")
	if string(code22[:4]) != "TSLA" {
		t.Fatalf("unexpected code22 buffer: %q", string(code22[:4]))
	}

	code23 := makeCode23("09988")
	if string(code23[:5]) != "09988" {
		t.Fatalf("unexpected code23 buffer: %q", string(code23[:5]))
	}

	file40 := makeFixed40("block.dat")
	if string(file40[:9]) != "block.dat" {
		t.Fatalf("unexpected file40 buffer: %q", string(file40[:9]))
	}

	file80 := makeFixed80("test.txt")
	if string(file80[:8]) != "test.txt" {
		t.Fatalf("unexpected file80 buffer: %q", string(file80[:8]))
	}

	file300 := makeFixed300("foo")
	if string(file300[:3]) != "foo" {
		t.Fatalf("unexpected file300 buffer: %q", string(file300[:3]))
	}

	file43 := makeFixed43("TSLA")
	if string(file43[:4]) != "TSLA" {
		t.Fatalf("unexpected file43 buffer: %q", string(file43[:4]))
	}
}

func TestQuotesSortReverse(t *testing.T) {
	if got := quotesSortReverse(SortCode, true); got != 0 {
		t.Fatalf("unexpected code sort reverse: %d", got)
	}
	if got := quotesSortReverse(SortPrice, false); got != 1 {
		t.Fatalf("unexpected asc sort reverse: %d", got)
	}
	if got := quotesSortReverse(SortPrice, true); got != 2 {
		t.Fatalf("unexpected desc sort reverse: %d", got)
	}
}

func TestMakeExStocks(t *testing.T) {
	stocks, err := makeExStocks([]uint8{ExCategoryUSStock, ExCategoryHKMainBoard}, []string{"TSLA", "09988"})
	if err != nil {
		t.Fatalf("makeExStocks failed: %v", err)
	}
	if len(stocks) != 2 {
		t.Fatalf("unexpected stock len: %d", len(stocks))
	}
	if stocks[0].Category != ExCategoryUSStock || stocks[0].Code != "TSLA" {
		t.Fatalf("unexpected first ex stock: %+v", stocks[0])
	}
}

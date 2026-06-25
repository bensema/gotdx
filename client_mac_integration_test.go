package gotdx

import (
	"testing"
	"time"

	"github.com/bensema/gotdx/types"
)

const macIntegrationHistoryDate = 20260430

func newMACIntegrationClient() *Client {
	primary, pool := defaultAddressAndPool(MACHostAddresses(), "")
	return NewMAC(
		WithMacTCPAddress(primary),
		WithMacTCPAddressPool(pool...),
		WithTimeoutSec(6),
	)
}

func Test_tdx_MACRecentProtocols(t *testing.T) {
	requireIntegration(t)

	t.Run("quotes_with_date", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		reply, err := client.MACQuotesWithDate(types.MarketSZ.Uint8(), "000001", macIntegrationHistoryDate)
		if err != nil {
			t.Fatalf("MACQuotesWithDate failed: %v", err)
		}
		if reply.Code != "000001" {
			t.Fatalf("unexpected quote code: %+v", reply)
		}
		if len(reply.ChartData) == 0 {
			t.Fatalf("expected historical chart data, got empty reply: %+v", reply)
		}
		if reply.DateTime.Format(time.DateTime) == "" {
			t.Fatalf("expected quote summary datetime, got empty reply: %+v", reply)
		}
	})

	t.Run("symbol_quotes", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		reply, err := client.MACSymbolQuotes(
			[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()},
			[]string{"000001", "600000"},
			DefaultMACSymbolQuotesFieldBitmap(),
		)
		if err != nil {
			t.Fatalf("MACSymbolQuotes failed: %v", err)
		}
		if len(reply.ActiveFields) == 0 {
			t.Fatalf("expected active dynamic fields, got empty reply: %+v", reply)
		}
		if len(reply.Stocks) == 0 {
			t.Fatalf("expected symbol quotes, got empty reply: %+v", reply)
		}
		if reply.Stocks[0].Symbol == "" {
			t.Fatalf("unexpected first symbol quote item: %+v", reply.Stocks[0])
		}
	})

	t.Run("board_members_quotes_dynamic", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		reply, err := client.MACBoardMembersQuotesDynamic("880761", 5, 14, 1, DefaultMACBoardMembersQuotesFieldBitmap())
		if err != nil {
			t.Fatalf("MACBoardMembersQuotesDynamic failed: %v", err)
		}
		if len(reply.ActiveFields) == 0 {
			t.Fatalf("expected active dynamic fields, got empty reply: %+v", reply)
		}
		if len(reply.Stocks) == 0 {
			t.Fatalf("expected board member quotes, got empty reply: %+v", reply)
		}
		if reply.Stocks[0].Symbol == "" {
			t.Fatalf("expected stock symbol in first item, got: %+v", reply.Stocks[0])
		}
	})

	t.Run("transactions_with_date", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		items, err := client.MACTransactionsWithDate(types.MarketSZ.Uint8(), "000001", 0, 20, macIntegrationHistoryDate)
		if err != nil {
			t.Fatalf("MACTransactionsWithDate failed: %v", err)
		}
		if len(items) == 0 {
			t.Fatalf("expected historical transactions, got empty result")
		}
		if items[0].Price <= 0 || items[0].Time == "" {
			t.Fatalf("unexpected first transaction: %+v", items[0])
		}
	})

	t.Run("auction", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		items, err := client.MACAuction(types.MarketSZ.Uint8(), "000001", 0, 20)
		if err != nil {
			t.Fatalf("MACAuction failed: %v", err)
		}
		if len(items) == 0 {
			t.Log("MAC auction returned no data; this can happen outside the auction window")
			return
		}
		if items[0].Time == "" {
			t.Fatalf("unexpected first auction item: %+v", items[0])
		}
	})

	t.Run("tick_charts", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		reply, err := client.MACTickCharts(types.MarketSZ.Uint8(), "000001", 0, 3)
		if err != nil {
			t.Fatalf("MACTickCharts failed: %v", err)
		}
		if reply.Code != "000001" || reply.DateTime.Format(time.DateTime) == "" {
			t.Fatalf("unexpected tick charts reply: %+v", reply)
		}
		if len(reply.Charts) == 0 || reply.Total == 0 {
			t.Fatalf("expected tick chart data, got empty reply: %+v", reply)
		}
		nonEmptyDays := 0
		for _, day := range reply.Charts {
			if len(day.Ticks) > 0 {
				nonEmptyDays++
			}
		}
		if nonEmptyDays == 0 {
			t.Fatalf("expected at least one non-empty tick chart day: %+v", reply.Charts)
		}
	})

	t.Run("symbol_info", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		reply, err := client.MACSymbolInfo(types.MarketSZ.Uint8(), "000001")
		if err != nil {
			t.Fatalf("MACSymbolInfo failed: %v", err)
		}
		if reply.Code != "000001" || reply.Name == "" {
			t.Fatalf("unexpected symbol info reply: %+v", reply)
		}
	})

	t.Run("capital_flow", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		reply, err := client.MACCapitalFlow(types.MarketSZ.Uint8(), "000001")
		if err != nil {
			t.Fatalf("MACCapitalFlow failed: %v", err)
		}
		if reply.QueryInfo == "" {
			t.Fatalf("expected capital flow query info, got empty reply: %+v", reply)
		}
	})

	t.Run("file_list_and_download", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		meta, err := client.MACFileList("StockInfo.dat", 0)
		if err != nil {
			t.Fatalf("MACFileList failed: %v", err)
		}
		if meta.Offset != 0 {
			t.Fatalf("unexpected file metadata: %+v", meta)
		}
		if meta.Size == 0 {
			t.Logf("MAC file metadata reports zero size for StockInfo.dat: %+v", meta)
			return
		}

		content, err := client.MACDownloadFullFile("StockInfo.dat", 1, meta.Size)
		if err != nil {
			t.Fatalf("MACDownloadFullFile failed: %v", err)
		}
		if len(content) == 0 {
			t.Fatalf("expected downloaded file content, got empty payload")
		}
	})

	t.Run("market_monitor", func(t *testing.T) {
		client := newMACIntegrationClient()
		defer client.Disconnect()

		items, err := client.MACMarketMonitor(types.MarketSZ.Uint8(), 0, 20)
		if err != nil {
			t.Fatalf("MACMarketMonitor failed: %v", err)
		}
		if len(items) == 0 {
			t.Log("MAC market monitor returned no data; this can happen outside trading hours")
			return
		}
		if items[0].Code == "" {
			t.Fatalf("unexpected first market monitor item: %+v", items[0])
		}
	})
}

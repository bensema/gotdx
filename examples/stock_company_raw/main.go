package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	categories, err := client.GetCompanyCategories(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("company_categories count=%d", categories.Count)

	if len(categories.Categories) > 0 {
		first := categories.Categories[0]
		content, err := client.GetCompanyContent(gotdx.MarketSZ, "000001", first.Filename, first.Start, first.Length)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("company_content name=%s filename=%s preview=%s",
			first.Name, first.Filename, preview(content.Content, 80))
	}

	finance, err := client.GetFinanceInfo(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("finance code=%s updated=%d total_shares=%.2f revenue=%.2f profit=%.2f",
		finance.Code, finance.UpdatedDate, finance.TotalShares, finance.OperatingRevenue, finance.NetProfit)

	xdxr, err := client.GetXDXRInfo(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("xdxr code=%s count=%d", xdxr.Code, xdxr.Count)
	for _, item := range xdxr.List[:min(5, len(xdxr.List))] {
		log.Printf("xdxr date=%s category=%d name=%s", item.Date.Format("2006-01-02"), item.Category, item.Name)
	}

	bundle, err := client.GetCompanyInfo(gotdx.MarketSZ, "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("company_bundle sections=%d xdxr=%d finance_nil=%t",
		len(bundle.Sections), len(bundle.XDXR), bundle.Finance == nil)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func preview(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

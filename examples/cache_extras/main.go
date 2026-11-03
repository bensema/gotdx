// Command cache_extras 演示 client_extras 族函数（表格文件、CSV 文件、聚合公司信息）
// 的缓存版调用：首次调用回源行情服务器并写入 `Redis`，二次调用直接命中缓存不再触碰
// 行情源；通过 `Stats` 观察命中/回源计数变化。
//
// 前置条件：本地 `Redis`（默认 `127.0.0.1:6379`）与主站网络可达。
// 运行示例：`go run ./examples/cache_extras`
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bensema/gotdx/cache"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
	"github.com/redis/go-redis/v9"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", DB: 2})
	defer rdb.Close()
	cached := cache.NewCacheClient(rdb, client)
	ctx := context.Background()

	// 竖线分隔表格文件 `tdxhy.cfg`（服务端静态配置，按日更新）：首次回源下载并解析，
	// 二次查询命中缓存；两轮之间可见 `Fetches` 与 `Hits` 各 +1。
	log.Println("== GetTableFile ==")
	rows, err := cached.GetTableFile(ctx, "tdxhy.cfg")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("table rows=%d preview=%v", len(rows), previewRow(rows, 0))
	if _, err := cached.GetTableFile(ctx, "tdxhy.cfg"); err != nil {
		log.Fatalln(err)
	}
	log.Printf("table hit again ok stats=%s", statsText(cached.Stats()))

	// CSV 文件 `spec/speckzzdata.txt`（证券代码字典）：同为静态文件整段缓存，
	// 键带 `csv` 操作标签，与竖线表格互不串键。
	log.Println("== GetCSVFile ==")
	csvRows, err := cached.GetCSVFile(ctx, "spec/speckzzdata.txt")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("csv rows=%d preview=%v", len(csvRows), previewRow(csvRows, 0))
	if _, err := cached.GetCSVFile(ctx, "spec/speckzzdata.txt"); err != nil {
		log.Fatalln(err)
	}
	log.Printf("csv hit again ok stats=%s", statsText(cached.Stats()))

	// 聚合公司信息（F10 章节 + 除权除息 + 财务）：聚合层单键缓存，命中时一次
	// `Redis` 往返即得整包结果。
	log.Println("== GetCompanyInfo ==")
	bundle, err := cached.GetCompanyInfo(ctx, types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("bundle sections=%d xdxr=%d finance_nil=%t",
		len(bundle.Sections), len(bundle.XDXR), bundle.Finance == nil)
	if _, err := cached.GetCompanyInfo(ctx, types.MarketSZ.Uint8(), "000001"); err != nil {
		log.Fatalln(err)
	}
	log.Printf("bundle hit again ok stats=%s", statsText(cached.Stats()))

	fmt.Printf("最终统计：命中=%d 空标记命中=%d 回源=%d 回源失败=%d 熔断拒绝=%d\n",
		cached.Stats().Hits, cached.Stats().EmptyHits, cached.Stats().Fetches,
		cached.Stats().SourceErrors, cached.Stats().CircuitRejects)
}

// previewRow 取二维表格首行转成日志可读文本，长度超限时截断。
//
// 参数 `rows` 为二维字符串表格；参数 `index` 为要预览的行号。首行可能含表头或大字段，
// 统一截断到 60 字符，避免日志过长。
func previewRow(rows [][]string, index int) []string {
	if len(rows) == 0 || index >= len(rows) {
		return nil
	}
	out := make([]string, 0, len(rows[index]))
	for _, cell := range rows[index] {
		if len(cell) > 60 {
			cell = cell[:60]
		}
		out = append(out, cell)
	}
	return out
}

// statsText 把缓存统计快照压成单行文本，便于日志对比两轮计数。
//
// 参数 `s` 为 `cache.Stats` 快照；返回 `命中/空标记命中/回源/回源失败/熔断拒绝` 五个计数。
func statsText(s cache.Stats) string {
	return fmt.Sprintf("hits=%d empty_hits=%d fetches=%d source_errors=%d circuit_rejects=%d",
		s.Hits, s.EmptyHits, s.Fetches, s.SourceErrors, s.CircuitRejects)
}

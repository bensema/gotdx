// Command stock_block_raw_cache 演示板块文件能力（`block.go` 导出方法）的缓存调用：
// 首次调用回源下载文件并解析、结果写入 `Redis`，二次调用直接命中缓存不再触碰行情源，
// 通过 `Stats` 观察命中/回源计数变化。原始接口调用见 `stock_block_raw`。
//
// 覆盖 `ParseBlockFlat`/`ParseBlockGroups` 对应的 `GetParsedBlockFile`/`GetGroupedBlockFile`
// 两族（缓存版见 `cache/block.go`）。
//
// 前置条件：本地 `Redis`（默认 `127.0.0.1:6379`）与主站网络可达。
// 运行示例：`go run ./examples/stock_block_raw_cache`
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

	// 缓存调用：首次回源下载并解析、写入 `Redis`，二次命中不再触碰行情源。
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379", DB: 2})
	defer rdb.Close()
	cached := cache.NewCacheClient(rdb, client)
	ctx := context.Background()

	log.Println("== 缓存调用 GetParsedBlockFile ==")
	cFlat, err := cached.GetParsedBlockFile(ctx, types.BlockFileGN)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("cached flat count=%d", len(cFlat))
	if _, err := cached.GetParsedBlockFile(ctx, types.BlockFileGN); err != nil {
		log.Fatalln(err)
	}
	log.Printf("flat hit again ok stats=%s", statsText(cached.Stats()))

	log.Println("== 缓存调用 GetGroupedBlockFile ==")
	cGrouped, err := cached.GetGroupedBlockFile(ctx, types.BlockFileFG)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("cached grouped count=%d", len(cGrouped))
	if _, err := cached.GetGroupedBlockFile(ctx, types.BlockFileFG); err != nil {
		log.Fatalln(err)
	}
	log.Printf("grouped hit again ok stats=%s", statsText(cached.Stats()))

	fmt.Printf("最终统计：命中=%d 空标记命中=%d 回源=%d 回源失败=%d 熔断拒绝=%d\n",
		cached.Stats().Hits, cached.Stats().EmptyHits, cached.Stats().Fetches,
		cached.Stats().SourceErrors, cached.Stats().CircuitRejects)
}

// min 返回整数 a 与 b 中较小的一个。
//
// 参数 a、b 为待比较的两个整数；返回值为二者较小值，用于在切片截断时把长度限制在有效范围内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// statsText 把缓存统计快照压成单行文本，便于日志对比两轮计数。
//
// 参数 `s` 为 `cache.Stats` 快照；返回 `命中/空标记命中/回源/回源失败/熔断拒绝` 五个计数。
func statsText(s cache.Stats) string {
	return fmt.Sprintf("hits=%d empty_hits=%d fetches=%d source_errors=%d circuit_rejects=%d",
		s.Hits, s.EmptyHits, s.Fetches, s.SourceErrors, s.CircuitRejects)
}

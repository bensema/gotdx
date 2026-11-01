// 文件 block.go 实现板块文件能力的缓存封装：板块文件为服务端静态内容，
// 下载并解析后的结果为整段非时序数据，适合键值 `String` 整段缓存
// （变化极慢，长 `TTL`）。

package cache

import (
	"context"
	"time"

	"github.com/bensema/gotdx"
)

// blockCacheTTL 为板块文件解析结果的缓存时长：服务端板块文件每日更新，
// 30 分钟兼顾新鲜度与命中率。
const blockCacheTTL = 30 * time.Minute

// GetParsedBlockFile 缓存版板块文件扁平解析。
//
// 方案：需要缓存 ✓（板块文件静态且每日更新）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存（无最新一根/历史之分，不适用 `ZSet`/`Hash`）。
// 键按文件名隔离并带操作标签，与 `GetGroupedBlockFile` 同形参数不串键；
// 未命中经 `kvGet` 回源（下载文件并 `ParseBlockFlat` 解析）后整段写入。
// 下载或解析失败写入失败空标记，后续查询直接返回 ErrSourceFailed。
func (c *CacheClient) GetParsedBlockFile(ctx context.Context, filename string) ([]gotdx.BlockFlatItem, error) {
	return c.kvGet(ctx, "block:parsed:"+filename, blockCacheTTL,
		func(ctx context.Context) ([]gotdx.BlockFlatItem, bool, error) {
			content, err := c.cli.GetBlockFile(filename)
			if err != nil {
				return nil, false, err
			}
			items, err := gotdx.ParseBlockFlat(content)
			if err != nil {
				return nil, false, err
			}
			return items, false, nil
		})
}

// GetGroupedBlockFile 缓存版板块文件分组解析。
//
// 方案：需要缓存 ✓（板块文件静态且每日更新）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键规则与 `GetParsedBlockFile` 一致，仅操作
// 标签不同；未命中经 `kvGet` 回源（下载文件并 `ParseBlockGroups` 解析）
// 后整段写入。下载或解析失败写入失败空标记。
func (c *CacheClient) GetGroupedBlockFile(ctx context.Context, filename string) ([]gotdx.BlockGroup, error) {
	return c.kvGet(ctx, "block:grouped:"+filename, blockCacheTTL,
		func(ctx context.Context) ([]gotdx.BlockGroup, bool, error) {
			content, err := c.cli.GetBlockFile(filename)
			if err != nil {
				return nil, false, err
			}
			groups, err := gotdx.ParseBlockGroups(content)
			if err != nil {
				return nil, false, err
			}
			return groups, false, nil
		})
}

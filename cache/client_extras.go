// 文件 client_extras.go 实现根包扩展文件能力与聚合公司信息的缓存封装：表格文件
// （竖线分隔/CSV）为服务端静态内容，解析结果为整段非时序数据；公司信息聚合为低频
// 变更的大对象。均适合键值 `String` 整段缓存（长 `TTL`，无最新一根/历史之分，
// 不适用 `ZSet`/`Hash`）。
//
// 存储方式说明：
//   - `GetTableFile`/`GetCSVFile` 返回二维字符串表格，整段 MessagePack 编码后存入
//     `Redis` `String`，键按文件名隔离并带操作标签（pipe/csv），同形参数不串键；
//     未命中经 `kvGet` 回源（下载并解析）后整段写入。
//   - `GetCompanyInfo` 返回聚合 Bundle（章节列表 + 除权除息 + 财务），底层组件
//     （分类目录/章节正文/除权除息/财务）虽已在 `client_company.go` 分别缓存，
//     但聚合层单键命中可把多次 Redis 往返收敛为一次，代价是与底层缓存双份存储；
//     公司信息低频变更，该代价可接受。
//   - 原始文件字节（`DownloadFullFile`/`GetBlockFile`）不在本文件缓存：块级
//     `DownloadFile` 已缓存（见 `client_company.go`），且板块文件解析结果已缓存
//     （见 `block.go`），再存字节层只增加存储不带来新命中率。

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/bensema/gotdx"
)

// tableCacheTTL 为表格文件解析结果的缓存时长：服务端此类文件按日更新，
// 30 分钟兼顾新鲜度与命中率（与板块文件同尺度）。
const tableCacheTTL = 30 * time.Minute

// GetTableFile 缓存版竖线分隔表格文件解析。
//
// 方案：需要缓存 ✓（表格文件静态且按日更新）；结果为整段非时序数据，采用键值
// `String` 整段缓存（无最新一根/历史之分，不适用 `ZSet`/`Hash`）。键按文件名隔离
// 并带 `pipe` 操作标签，与 `GetCSVFile` 同形参数不串键；未命中经 `kvGet` 回源
// （下载完整文件并解析竖线表格）后整段写入。下载或解析失败写入失败空标记，后续
// 查询直接返回 `ErrSourceFailed`。
func (c *CacheClient) GetTableFile(ctx context.Context, filename string) ([][]string, error) {
	return c.kvGet(ctx, "table:pipe:"+filename, tableCacheTTL,
		func(ctx context.Context) ([][]string, bool, error) {
			rows, err := c.cli.GetTableFile(filename)
			if err != nil {
				return nil, false, err
			}
			return rows, false, nil
		})
}

// GetCSVFile 缓存版 CSV 文件解析。
//
// 方案：需要缓存 ✓（CSV 文件静态且按日更新）；结果为整段非时序数据，采用键值
// `String` 整段缓存。键按文件名隔离并带 `csv` 操作标签，与 `GetTableFile` 同形
// 参数不串键；未命中经 `kvGet` 回源（下载完整文件并解析 CSV）后整段写入。下载或
// 解析失败写入失败空标记。
func (c *CacheClient) GetCSVFile(ctx context.Context, filename string) ([][]string, error) {
	return c.kvGet(ctx, "table:csv:"+filename, tableCacheTTL,
		func(ctx context.Context) ([][]string, bool, error) {
			rows, err := c.cli.GetCSVFile(filename)
			if err != nil {
				return nil, false, err
			}
			return rows, false, nil
		})
}

// GetCompanyInfo 缓存版聚合公司信息查询（F10 章节 + 除权除息 + 财务）。
//
// 方案：需要缓存 ✓（公司信息低频变更）；结果为聚合 Bundle（含章节全文，体积大），
// 采用键值 `String` 整段缓存。键按市场与证券代码隔离；未命中经 `kvGet` 回源
// （`c.cli.GetCompanyInfo` 聚合拉取）后整段写入。聚合层单键命中省去底层组件多次
// Redis 往返；回源失败写入失败空标记，后续查询直接返回 `ErrSourceFailed`。
func (c *CacheClient) GetCompanyInfo(ctx context.Context, market uint8, code string) (*gotdx.CompanyInfoBundle, error) {
	key := fmt.Sprintf("company:bundle:%d:%s", market, code)
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*gotdx.CompanyInfoBundle, bool, error) {
			bundle, err := c.cli.GetCompanyInfo(market, code)
			if err != nil {
				return nil, false, err
			}
			return bundle, false, nil
		})
}

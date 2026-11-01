// 文件 client_exquote.go 实现扩展行情族能力的缓存封装：服务器信息、计数与分类、
// 证券列表、行情快照、K 线与分时、历史逐笔、实验接口、板块与映射、扩展站文件及
// 表格查询均为整段非时序数据，适合键值 `String` 整段缓存（无最新一根/历史之分，
// 不适用 `ZSet`/`Hash`）。
//
// 按实时性分两档 `TTL`：行情快照与列表类使用秒级短 `TTL` 兼顾新鲜度，K 线、分时、
// 静态数据与文件类使用分钟级 `TTL`。键以 `ex:`（行情族）与 `exfile:`（扩展站文件）
// 前缀隔离，与主站文件键 `file:` 不串数据；未命中统一经 `kvGet` 回源（内部含
// `singleflight` 合并、分布式锁防击穿、空标记防穿透与熔断保护）。

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/bensema/gotdx/proto"
)

// exQuoteTTL 为行情快照与列表类数据的缓存时长：实时行情秒级变化，5 秒
// 兼顾新鲜度与重复查询命中。
const exQuoteTTL = 5 * time.Second

// exStaticTTL 为 K 线、分时、静态数据与文件类内容的缓存时长：此类数据分钟级
// 变化或几乎不变，5 分钟兼顾新鲜度与命中率。
const exStaticTTL = 5 * time.Minute

// GetExServerInfo 缓存版扩展行情服务器信息查询。
//
// 方案：需要缓存 ✓（服务器信息几乎不变）；结果为整段非时序数据，采用键值
// `String` 整段缓存。未命中经 `kvGet` 回源（`c.cli.GetExServerInfo`）后整段
// 写入。回源失败写入失败空标记。
func (c *CacheClient) GetExServerInfo(ctx context.Context) (*proto.ExServerInfoReply, error) {
	const key = "ex:server"
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExServerInfoReply, bool, error) {
			reply, err := c.cli.GetExServerInfo()
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetCount 缓存版扩展市场证券计数查询。
//
// 方案：需要缓存 ✓（总计数随交易日缓慢变化）；结果为整段非时序数据，采用键值
// `String` 整段缓存。未命中经 `kvGet` 回源（`c.cli.ExGetCount`）后整段写入。
// 回源失败写入失败空标记。
func (c *CacheClient) ExGetCount(ctx context.Context) (*proto.ExGetCountReply, error) {
	const key = "ex:count"
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetCountReply, bool, error) {
			reply, err := c.cli.ExGetCount()
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetCategoryList 缓存版扩展市场分类列表查询。
//
// 方案：需要缓存 ✓（分类结构低频变化）；结果为整段非时序数据，采用键值 `String`
// 整段缓存。未命中经 `kvGet` 回源（`c.cli.ExGetCategoryList`）后整段写入。
// 回源失败写入失败空标记。
func (c *CacheClient) ExGetCategoryList(ctx context.Context) (*proto.ExGetCategoryListReply, error) {
	const key = "ex:cats"
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetCategoryListReply, bool, error) {
			reply, err := c.cli.ExGetCategoryList()
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetList 缓存版扩展市场证券列表查询。
//
// 方案：需要缓存 ✓（列表按页查询、重复翻页命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按起始偏移与条数隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetList`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetList(ctx context.Context, start uint32, count uint16) (*proto.ExGetListReply, error) {
	key := fmt.Sprintf("ex:list:%d:%d", start, count)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetListReply, bool, error) {
			reply, err := c.cli.ExGetList(start, count)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetListExtra 缓存版扩展市场证券列表扩展查询。
//
// 方案：需要缓存 ✓（列表按页查询、重复翻页命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按三个分页参数隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetListExtra`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetListExtra(ctx context.Context, a uint16, b uint16, count uint16) (*proto.ExGetListExtraReply, error) {
	key := fmt.Sprintf("ex:listextra:%d:%d:%d", a, b, count)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetListExtraReply, bool, error) {
			reply, err := c.cli.ExGetListExtra(a, b, count)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetQuotesList 缓存版扩展市场行情列表查询。
//
// 方案：需要缓存 ✓（列表按页查询、重复翻页命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按类别、分页与排序参数隔离；未命中经 `kvGet`
// 回源（`c.cli.ExGetQuotesList`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetQuotesList(ctx context.Context, category uint8, start uint16, count uint16, sortType uint16, reverse bool) (*proto.ExGetQuotesListReply, error) {
	key := fmt.Sprintf("ex:quoteslist:%d:%d:%d:%d:%t", category, start, count, sortType, reverse)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetQuotesListReply, bool, error) {
			reply, err := c.cli.ExGetQuotesList(category, start, count, sortType, reverse)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetQuote 缓存版扩展市场单品种行情快照查询。
//
// 方案：需要缓存 ✓（快照秒级变化，重复查询命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按类别与标的代码隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetQuote`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetQuote(ctx context.Context, category uint8, code string) (*proto.ExGetQuoteReply, error) {
	key := fmt.Sprintf("ex:quote:%d:%s", category, code)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetQuoteReply, bool, error) {
			reply, err := c.cli.ExGetQuote(category, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetQuotes 缓存版扩展市场批量行情快照查询。
//
// 方案：需要缓存 ✓（快照秒级变化，同一批次重复查询命中率高）；结果为整段非时序
// 数据，采用键值 `String` 整段缓存。键按类别与代码切片拼接隔离；未命中经
// `kvGet` 回源（`c.cli.ExGetQuotes`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetQuotes(ctx context.Context, categories []uint8, codes []string) (*proto.ExGetQuotesReply, error) {
	key := fmt.Sprintf("ex:quotes:%v:%v", categories, codes)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetQuotesReply, bool, error) {
			reply, err := c.cli.ExGetQuotes(categories, codes)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetQuotes2 缓存版扩展市场批量行情快照查询（第二种批量接口）。
//
// 方案：需要缓存 ✓（快照秒级变化，同一批次重复查询命中率高）；结果为整段非时序
// 数据，采用键值 `String` 整段缓存。键按类别与代码切片拼接隔离；未命中经
// `kvGet` 回源（`c.cli.ExGetQuotes2`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetQuotes2(ctx context.Context, categories []uint8, codes []string) (*proto.ExGetQuotesReply, error) {
	key := fmt.Sprintf("ex:quotes2:%v:%v", categories, codes)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetQuotesReply, bool, error) {
			reply, err := c.cli.ExGetQuotes2(categories, codes)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetKLine 缓存版扩展市场 K 线查询。
//
// 方案：需要缓存 ✓（历史段几乎不变，重复拉取命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按类别、代码、周期与区间参数隔离；未命中经
// `kvGet` 回源（`c.cli.ExGetKLine`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetKLine(ctx context.Context, category uint8, code string, period uint16, start uint32, count uint16, times uint16) (*proto.ExGetKLineReply, error) {
	key := fmt.Sprintf("ex:kline:%d:%s:%d:%d:%d:%d", category, code, period, start, count, times)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetKLineReply, bool, error) {
			reply, err := c.cli.ExGetKLine(category, code, period, start, count, times)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetExperiment2487 缓存版扩展市场实验接口 2487 查询。
//
// 方案：需要缓存 ✓（实验接口返回低频数据）；结果为整段非时序数据，采用键值
// `String` 整段缓存。键按类别与标的代码隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetExperiment2487`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetExperiment2487(ctx context.Context, category uint8, code string) (*proto.ExExperiment2487Reply, error) {
	key := fmt.Sprintf("ex:exp2487:%d:%s", category, code)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExExperiment2487Reply, bool, error) {
			reply, err := c.cli.ExGetExperiment2487(category, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetExperiment2488 缓存版扩展市场实验接口 2488 查询。
//
// 方案：需要缓存 ✓（实验接口返回低频数据）；结果为整段非时序数据，采用键值
// `String` 整段缓存。键按类别、标的代码与模式隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetExperiment2488`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetExperiment2488(ctx context.Context, category uint8, code string, mode uint16) (*proto.ExExperiment2488Reply, error) {
	key := fmt.Sprintf("ex:exp2488:%d:%s:%d", category, code, mode)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExExperiment2488Reply, bool, error) {
			reply, err := c.cli.ExGetExperiment2488(category, code, mode)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetKLine2 缓存版扩展市场 K 线查询（长区间变体）。
//
// 方案：需要缓存 ✓（历史段几乎不变，重复拉取命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按类别、代码、周期与区间参数隔离；未命中经
// `kvGet` 回源（`c.cli.ExGetKLine2`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetKLine2(ctx context.Context, category uint8, code string, period uint16, start uint32, count uint32, times uint16) (*proto.ExGetKLine2Reply, error) {
	key := fmt.Sprintf("ex:kline2:%d:%s:%d:%d:%d:%d", category, code, period, start, count, times)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetKLine2Reply, bool, error) {
			reply, err := c.cli.ExGetKLine2(category, code, period, start, count, times)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetHistoryTransaction 缓存版扩展市场历史逐笔成交查询。
//
// 方案：需要缓存 ✓（历史逐笔永久不变，重复查询命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按日期、类别与标的代码隔离；未命中经 `kvGet`
// 回源（`c.cli.ExGetHistoryTransaction`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetHistoryTransaction(ctx context.Context, date uint32, category uint8, code string) (*proto.ExGetHistoryTransactionReply, error) {
	key := fmt.Sprintf("ex:histtrans:%d:%d:%s", date, category, code)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetHistoryTransactionReply, bool, error) {
			reply, err := c.cli.ExGetHistoryTransaction(date, category, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetTickChart 缓存版扩展市场分时图查询。
//
// 方案：需要缓存 ✓（历史分时点不变，重复查询命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按类别与标的代码隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetTickChart`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetTickChart(ctx context.Context, category uint8, code string) (*proto.ExGetTickChartReply, error) {
	key := fmt.Sprintf("ex:tick:%d:%s", category, code)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetTickChartReply, bool, error) {
			reply, err := c.cli.ExGetTickChart(category, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetHistoryTickChart 缓存版扩展市场历史分时图查询。
//
// 方案：需要缓存 ✓（历史分时永久不变，重复查询命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按日期、类别与标的代码隔离；未命中经 `kvGet`
// 回源（`c.cli.ExGetHistoryTickChart`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetHistoryTickChart(ctx context.Context, date uint32, category uint8, code string) (*proto.ExGetHistoryTickChartReply, error) {
	key := fmt.Sprintf("ex:histtick:%d:%d:%s", date, category, code)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetHistoryTickChartReply, bool, error) {
			reply, err := c.cli.ExGetHistoryTickChart(date, category, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetChartSampling 缓存版扩展市场图表采样查询。
//
// 方案：需要缓存 ✓（采样点秒级变化，重复查询命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按类别与标的代码隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetChartSampling`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetChartSampling(ctx context.Context, category uint8, code string) (*proto.ExGetChartSamplingReply, error) {
	key := fmt.Sprintf("ex:sampling:%d:%s", category, code)
	return c.kvGet(ctx, key, exQuoteTTL,
		func(ctx context.Context) (*proto.ExGetChartSamplingReply, bool, error) {
			reply, err := c.cli.ExGetChartSampling(category, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetBoardList 缓存版扩展市场板块列表查询。
//
// 方案：需要缓存 ✓（板块结构低频变化，重复翻页命中率高）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按板块类型与分页参数隔离；未命中经 `kvGet` 回源
// （`c.cli.ExGetBoardList`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetBoardList(ctx context.Context, boardType uint16, start uint16, pageSize uint16) (*proto.ExGetBoardListReply, error) {
	key := fmt.Sprintf("ex:board:%d:%d:%d", boardType, start, pageSize)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExGetBoardListReply, bool, error) {
			reply, err := c.cli.ExGetBoardList(boardType, start, pageSize)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetMapping2562 缓存版扩展市场映射接口 2562 查询。
//
// 方案：需要缓存 ✓（映射关系低频变化，重复分页查询命中率高）；结果为整段非时序
// 数据，采用键值 `String` 整段缓存。键按市场与区间参数隔离；未命中经 `kvGet`
// 回源（`c.cli.ExGetMapping2562`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetMapping2562(ctx context.Context, market uint16, start uint32, count uint32) (*proto.ExMapping2562Reply, error) {
	key := fmt.Sprintf("ex:mapping:%d:%d:%d", market, start, count)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.ExMapping2562Reply, bool, error) {
			reply, err := c.cli.ExGetMapping2562(market, start, count)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExGetFileMeta 缓存版扩展站文件元信息查询。
//
// 方案：需要缓存 ✓（文件大小与哈希值在文件更新前不变）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键以 `exfile:` 前缀隔离扩展站文件；未命中经
// `kvGet` 回源（`c.cli.ExGetFileMeta`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExGetFileMeta(ctx context.Context, filename string) (*proto.GetFileMetaReply, error) {
	key := "exfile:meta:" + filename
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.GetFileMetaReply, bool, error) {
			reply, err := c.cli.ExGetFileMeta(filename)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExDownloadFile 缓存版扩展站文件分块下载。
//
// 方案：需要缓存 ✓（文件内容在更新前不变，全量拉取时按块重复请求命中率高）；
// 结果为整段非时序数据，采用键值 `String` 整段缓存。键按文件名与字节区间隔离，
// 前缀 `exfile:` 隔离扩展站文件；未命中经 `kvGet` 回源（`c.cli.ExDownloadFile`）
// 后整段写入。回源失败写入失败空标记。
func (c *CacheClient) ExDownloadFile(ctx context.Context, filename string, start uint32, size uint32) (*proto.DownloadFileReply, error) {
	key := fmt.Sprintf("exfile:chunk:%s:%d:%d", filename, start, size)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (*proto.DownloadFileReply, bool, error) {
			reply, err := c.cli.ExDownloadFile(filename, start, size)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// ExDownloadFullFile 缓存版扩展站文件整包下载。
//
// 方案：需要缓存 ✓（文件内容在更新前不变，重复整包拉取命中率高）；结果为整段
// 非时序字节流，采用键值 `String` 整段缓存。键按文件名与目标字节数隔离；未命中
// 经 `kvGet` 回源（`c.cli.ExDownloadFullFile`）后整段写入。回源失败写入失败
// 空标记。
func (c *CacheClient) ExDownloadFullFile(ctx context.Context, filename string, size uint32) ([]byte, error) {
	key := fmt.Sprintf("exfile:full:%s:%d", filename, size)
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) ([]byte, bool, error) {
			content, err := c.cli.ExDownloadFullFile(filename, size)
			if err != nil {
				return nil, false, err
			}
			return content, false, nil
		})
}

// ExGetTable 缓存版扩展站表格查询。
//
// 方案：需要缓存 ✓（表格内容低频变化）；结果为整段非时序文本，采用键值 `String`
// 整段缓存。未命中经 `kvGet` 回源（`c.cli.ExGetTable`）后整段写入。回源失败
// 写入失败空标记。
func (c *CacheClient) ExGetTable(ctx context.Context) (string, error) {
	const key = "ex:table"
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (string, bool, error) {
			table, err := c.cli.ExGetTable()
			if err != nil {
				return "", false, err
			}
			return table, false, nil
		})
}

// ExGetTableDetail 缓存版扩展站表格详情查询。
//
// 方案：需要缓存 ✓（表格内容低频变化）；结果为整段非时序文本，采用键值 `String`
// 整段缓存。未命中经 `kvGet` 回源（`c.cli.ExGetTableDetail`）后整段写入。回源
// 失败写入失败空标记。
func (c *CacheClient) ExGetTableDetail(ctx context.Context) (string, error) {
	const key = "ex:tabledetail"
	return c.kvGet(ctx, key, exStaticTTL,
		func(ctx context.Context) (string, bool, error) {
			table, err := c.cli.ExGetTableDetail()
			if err != nil {
				return "", false, err
			}
			return table, false, nil
		})
}

// 文件 client_company.go 实现公司资料族与主站文件能力的缓存封装：分类目录、章节
// 正文、财务概况、除权除息与文件元信息、分块下载均为低频变更的整段非时序数据，
// 适合键值 `String` 整段缓存（长 `TTL`，无最新一根/历史之分，不适用 `ZSet`/`Hash`）。
//
// 键按市场、证券代码、文件名与字节区间隔离，操作间不串键；未命中统一经 `kvGet`
// 回源（内部含 `singleflight` 合并、分布式锁防击穿、空标记防穿透与熔断保护），
// 回源成功后整段写入并续 `TTL`。

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/bensema/gotdx/proto"
)

// companyCacheTTL 为公司资料与主站文件的缓存时长：服务端此类内容按日甚至按季度
// 更新，1 小时兼顾新鲜度与命中率。
const companyCacheTTL = 1 * time.Hour

// GetCompanyCategories 缓存版公司资料分类目录查询。
//
// 方案：需要缓存 ✓（分类目录变化极慢）；结果为整段非时序数据，采用键值 `String`
// 整段缓存（不适用 `ZSet`/`Hash`）。键按市场与证券代码隔离；未命中经 `kvGet`
// 回源（`c.cli.GetCompanyCategories`）后整段写入。回源失败写入失败空标记，后续
// 查询直接返回 `ErrSourceFailed`。
func (c *CacheClient) GetCompanyCategories(ctx context.Context, market uint8, code string) (*proto.GetCompanyCategoryReply, error) {
	key := fmt.Sprintf("company:cats:%d:%s", market, code)
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*proto.GetCompanyCategoryReply, bool, error) {
			reply, err := c.cli.GetCompanyCategories(market, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// GetCompanyContent 缓存版公司资料章节正文查询。
//
// 方案：需要缓存 ✓（正文内容静态）；结果为整段非时序数据，采用键值 `String`
// 整段缓存。键按市场、证券代码、文件名与正文字节区间隔离，同一证券不同章节
// 各占独立键；未命中经 `kvGet` 回源（`c.cli.GetCompanyContent`）后整段写入。
// 回源失败写入失败空标记。
func (c *CacheClient) GetCompanyContent(ctx context.Context, market uint8, code string, filename string, start uint32, length uint32) (*proto.GetCompanyContentReply, error) {
	key := fmt.Sprintf("company:content:%d:%s:%s:%d:%d", market, code, filename, start, length)
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*proto.GetCompanyContentReply, bool, error) {
			reply, err := c.cli.GetCompanyContent(market, code, filename, start, length)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// GetFinanceInfo 缓存版证券财务概况查询。
//
// 方案：需要缓存 ✓（财务数据按报告期更新，变化慢）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按市场与证券代码隔离；未命中经 `kvGet` 回源
// （`c.cli.GetFinanceInfo`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) GetFinanceInfo(ctx context.Context, market uint8, code string) (*proto.GetFinanceInfoReply, error) {
	key := fmt.Sprintf("company:finance:%d:%s", market, code)
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*proto.GetFinanceInfoReply, bool, error) {
			reply, err := c.cli.GetFinanceInfo(market, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// GetXDXRInfo 缓存版除权除息历史查询。
//
// 方案：需要缓存 ✓（权益变动记录低频追加，历史几乎不变）；结果为整段非时序
// 数据，采用键值 `String` 整段缓存。键按市场与证券代码隔离；未命中经 `kvGet`
// 回源（`c.cli.GetXDXRInfo`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) GetXDXRInfo(ctx context.Context, market uint8, code string) (*proto.GetXDXRInfoReply, error) {
	key := fmt.Sprintf("company:xdxr:%d:%s", market, code)
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*proto.GetXDXRInfoReply, bool, error) {
			reply, err := c.cli.GetXDXRInfo(market, code)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// GetFileMeta 缓存版主站文件元信息查询。
//
// 方案：需要缓存 ✓（文件大小与哈希值在文件更新前不变）；结果为整段非时序数据，
// 采用键值 `String` 整段缓存。键按文件名隔离；未命中经 `kvGet` 回源
// （`c.cli.GetFileMeta`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) GetFileMeta(ctx context.Context, filename string) (*proto.GetFileMetaReply, error) {
	key := "file:meta:" + filename
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*proto.GetFileMetaReply, bool, error) {
			reply, err := c.cli.GetFileMeta(filename)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

// DownloadFile 缓存版主站文件分块下载。
//
// 方案：需要缓存 ✓（文件内容在更新前不变，全量拉取时按块重复请求命中率高）；
// 结果为整段非时序数据，采用键值 `String` 整段缓存。键按文件名与字节区间
// （`start`/`size`）隔离，同一文件不同分块各占独立键；未命中经 `kvGet` 回源
// （`c.cli.DownloadFile`）后整段写入。回源失败写入失败空标记。
func (c *CacheClient) DownloadFile(ctx context.Context, filename string, start uint32, size uint32) (*proto.DownloadFileReply, error) {
	key := fmt.Sprintf("file:chunk:%s:%d:%d", filename, start, size)
	return c.kvGet(ctx, key, companyCacheTTL,
		func(ctx context.Context) (*proto.DownloadFileReply, bool, error) {
			reply, err := c.cli.DownloadFile(filename, start, size)
			if err != nil {
				return nil, false, err
			}
			return reply, false, nil
		})
}

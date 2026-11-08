// 文件 client_line.go 实现 gotdx 统一客户端（根目录 client_unified.go）
// 中能力的缓存封装：当前覆盖 StockKLine（K 线），并抽出可复用的位置分页
// 缓存工厂，供后续同族接口直接接入。
//
// # 方案
//
// StockKLine 的 start 是 offset（0 为最新端、增大向更早历史），count 是
// limit，返回从早到晚。缓存按「`ZSet` 闭合历史 + `String` 实时槽未闭合
// 最新一根」实现，存储位置即 offset：
//
//   - 未闭合的最新一根（全局 index 0）存实时槽。K 线的同一根在盘中会
//     持续演化（时间分数不变、开高低收在变），而基座 liveScript 对同分
//     是「忽略」，故本文件为这类数据单独定义 positionLiveScript：同分
//     覆盖、跨分时把前一根原子流转进 `ZSet`，保证「最新一根每次刷新都
//     取到新值」而不是停在该分钟的第一笔快照上。
//   - 已闭合的历史根按 Unix 毫秒分数存 `ZSet`，成员为单根 MessagePack
//     编码后再 base64 的文本（与基座 encodeMember 一致）；永久保留，不
//     自动淘汰。整根 `proto.SecurityBar` 全字段编码（含 Vol/Amount、
//     DateTime、UpCount/DownCount），读回时 msgpack 解码无损还原，不落
//     JSON 文本、不做字段裁剪。
//   - 实时槽值同样是 MessagePack：`分数|更新时刻|base64(msgpack)`，由
//     encodeLiveValue 拼装、decodeLiveValue 还原（同基座格式）。
//   - 查询时基座 tsQuery 把实时槽与 `ZSet` 合并、按分数降序（新→旧）
//     返回窗口，能力层再用 tsReverse 还原成业务要求的「从早到晚」。因此
//     start=0 的语义天然等价于「(0,1) 实时最新 + (1,count-1) 历史」两次
//     取数的合并，无需在能力层拆成两次请求。
//
// # 工厂与复用
//
// `newPositionPager` 是本文件的工厂方法：它把「从最新端倒数 offset」型
// 接口的整套缓存行为（实时槽新鲜度、`ZSet` 连续前缀、命中判定、深度
// 判据、singleflight、分布式锁、降级兜底）打包成一个泛型 pager。接入一个
// 新方法只需要提供五项配置，不必重复实现上面任何一条：
//
//  1. key：语义键（必须包含全部会影响数据的请求参数，避免串用）；
//  2. fetch(ctx, start, count)：按原窗口回源，返回从旧到新；
//  3. head(ctx)：取最新端若干条（含未闭合的那一条），返回从旧到新；
//  4. score(item)：单条的排序分数，必须随「越新」单调递增（K 线用 Unix 毫秒）；
//  5. headN / fresh：最新端保留条数、实时槽新鲜度阈值。
//
// 适用范围：K 线族（`StockKLine`、`StockKLineOffset`、`ExKLine`、
// `ExKLine2`）、分时点序列（`StockTickChart`、`ExTickChart`）、按时间
// 递增的历史逐笔（`StockHistoryTransaction`）等「start 为 offset、count
// 为 limit、index 0 为最新端、单条带单调时间分数」的接口。
//
// 不适用：无时间分数的列表快照（`StockList`、`StockQuotesList`、`ExList`，
// 整页内容随排序规则整体变化，缓存位置没有意义）；以及同一时刻可能有多条
// 的逐笔追加型接口（分数会撞车，要按 `uniqueScore=false` 追加写入，与
// 「同分覆盖」语义相反）。
//
// # 位置不变量
//
// `ZSet` 的位置只有在下面两条同时成立时才等于全局 offset：
//
//  1. 最新端连续：`ZSet` 最大分数 = 实时槽内那一根的前一根。由每次查询
//     先刷新实时槽（跨分时脚本原子流转前一根）维护。
//  2. 缓存深度连续：`ZSet` 的根数 + 实时槽 = 从最新端起的连续前缀长度，
//     中间没有洞。由「只在缓存已覆盖窗口起点之前时才写入」保证。
//
// 不用「相邻两根分数差 == 一个周期」检测缺口：跨交易日（15:00 → 次日
// 09:31）会天天误判，且节假日、半日市都无法用固定周期表达。
//
// # 回源与降级
//
// 一律按原窗口 (start,count) 回源，不预先拉取 (0, start+count)：深 offset
// 也只发一次正常条数的请求，不会出现注定失败的超大请求（源端对过大的
// 整段请求会返回 invalid kline item，还会污染熔断的连续失败计数）。
// 回源结果只在「缓存已覆盖窗口起点之前」（depth >= start）时才并入缓存，
// 写入因此始终是从最新端起的连续前缀；跳页（depth < start）只直通返回、
// 不写缓存，避免在 `ZSet` 中间留下洞造成位置错位。
//
// # 取舍
//
//   - 换手率：缓存的是 StockKLine 加工后的根（含按流通股本算出的
//     Turnover）。命中后不再重算，股本变更后历史根仍为旧值；重算需要
//     loadFloatShares，而它在 gotdx 侧未导出、cache 包不可见，故此处按
//     「尽力而为」口径接受旧值。
//   - 09:30 首根：1min/EXHQ_1MIN 由 StockKLine 在回源时就地补齐，缓存
//     的是补齐后的序列，位置与实时调用一致；若某次回源逐笔失败导致该
//     根缺失，影响面限于最新一根与当次回源窗口。
//   - start=0 的查询仍可能回源一次（实时槽超过 klineLiveRefresh 未更新），
//     这是「最新一根必须是实时数据」的必然代价，不是缺陷；start>0 的翻页
//     不因新鲜度回源，只在实时槽缺失（会导致位置基准前移一位）时补一次。
//   - 实时槽 `TTL` 取基座 DefaultLiveTTL（1 小时）。分钟级无影响；日线/
//     周线在「次日第一根出现」之前槽可能先过期，那根不进入 `ZSet`，只是
//     下次查询多一次回源重建，数据仍正确。长周期场景可上调 WithLiveTTL。
package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/bensema/gotdx/proto"
	"github.com/redis/go-redis/v9"
)

const (
	// klineLiveRefresh 为实时槽的新鲜度阈值：超过该时长未更新即回源
	// 刷新一次。K 线最新一根按分钟推进，盘中数值持续变化，5 秒兼顾
	// 实时性与行情源压力。
	klineLiveRefresh = 5 * time.Second
	// klineLiveFetch 为刷新实时槽时的回源条数：取两根，最新根写槽、
	// 次新根幂等写 `ZSet`，同时维护「最新端连续」不变量。
	klineLiveFetch = 2
)

// positionLiveScriptSrc 为位置分页型数据的实时槽原子更新脚本，与基座
// liveScript 的唯一差异是同分语义：
//
//   - 旧值分数大于新分数：新值乱序或过期，忽略（与基座一致）。
//   - 旧值分数等于新分数：同一根数据在盘中演化，直接覆盖槽值，
//     不把旧值流转进 `ZSet`（基座对同分是忽略，会导致最新一条永远停在
//     该周期的首笔快照上）。
//   - 旧值分数小于新分数：旧值那一条已闭合，先按分数幂等覆盖写入
//     `ZSet`（同分先删后加，避免重复补齐时堆积），再覆盖槽值。
//
// 参数：KEYS[1]=实时槽键，KEYS[2]=`ZSet` 键；ARGV[1]=新值（分数|更新
// 时刻|base64 数据），ARGV[2]=新分数文本，ARGV[3]=`TTL` 秒数。返回 1
// 表示更新生效，0 表示被忽略。
const positionLiveScriptSrc = `
local old = redis.call('GET', KEYS[1])
if old then
  local sep1 = string.find(old, '|')
  if sep1 then
    local oldscore = tonumber(string.sub(old, 1, sep1 - 1))
    local newscore = tonumber(ARGV[2])
    if oldscore > newscore then
      return 0
    end
    if oldscore < newscore then
      local sep2 = string.find(old, '|', sep1 + 1)
      if sep2 then
        redis.call('ZREMRANGEBYSCORE', KEYS[2], oldscore, oldscore)
        redis.call('ZADD', KEYS[2], oldscore, string.sub(old, sep2 + 1))
      end
    end
  end
end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[3])
return 1`

// positionLiveScript 为位置分页型实时槽原子更新的封装脚本。
var positionLiveScript = redis.NewScript(positionLiveScriptSrc)

// klineCacheKey 返回 K 线能力的语义键：按周期类别、市场、代码、周期倍数
// 与复权方式隔离，避免不同请求参数串用同一份缓存。
//
// times 为零时按 1 归一，与协议层「零值按一处理」保持一致，防止 0/1
// 分裂成两个键。
func klineCacheKey(category uint16, market uint8, code string, times, adjust uint16) string {
	if times == 0 {
		times = 1
	}
	return fmt.Sprintf("kline:%d:%d:%s:%d:%d", category, market, code, times, adjust)
}

// StockKLine 缓存版 K 线查询。
//
// 方案：需要缓存 ✓（历史 K 线已闭合后不再变化）；未闭合的最新一根存
// 实时槽并按 klineLiveRefresh 刷新，已闭合根存 `ZSet`，查询时由基座
// 合并两者、按分数降序给出窗口，再反转成从早到晚返回。位置即 offset：
// start=0 等价于「(0,1) 实时最新 + (1,count-1) 历史」的合并结果。
//
// 本方法只做参数到 pager 的映射，缓存行为全部由 positionPager 承担；
// 后续同类接口按同样方式接入即可，详见文件头「工厂与复用」。
//
// 参数含义与 gotdx.Client.StockKLine 完全一致：category 为 `types.KLINE_TYPE_*`，
// market 为市场代码，code 为证券代码，start 为起始偏移条数（从 0 开始，
// 0 表示最新端），count 为请求条数，times 为周期倍数（零值按 1），adjust
// 为复权方式。命中缓存时不触碰行情源。
func (c *CacheClient) StockKLine(category uint16, market uint8, code string,
	start uint16, count uint16, times uint16, adjust uint16) ([]proto.SecurityBar, error) {
	// 空窗口不触网：与原始接口对 count=0 的返回（空切片、无错误）一致。
	if count == 0 {
		return []proto.SecurityBar{}, nil
	}
	p := c.newPositionPager(
		klineCacheKey(category, market, code, times, adjust),
		// fetch：按原窗口回源（裸调用，熔断与统计由 pager 统一包装）。
		func(ctx context.Context, start, count int) ([]proto.SecurityBar, error) {
			return c.cli.StockKLine(category, market, code, uint16(start), uint16(count), times, adjust)
		},
		// head：取最新端两根，最新根进槽、次新根进 `ZSet`。
		func(ctx context.Context) ([]proto.SecurityBar, error) {
			return c.cli.StockKLine(category, market, code, 0, klineLiveFetch, times, adjust)
		},
		int(klineLiveFetch),
		// score：越新分数越大（Unix 毫秒）。
		func(b proto.SecurityBar) float64 { return float64(b.DateTime.UnixMilli()) },
		klineLiveRefresh,
	)
	ctx := context.Background()
	items, err := p.query(ctx, int(start), int(count))
	if err != nil {
		return nil, err
	}
	return items, nil
}

// indexBarCacheKey 返回指数 K 线能力的语义键：按周期类别、市场、代码隔离，
// 避免不同请求参数串用同一份缓存。
//
// gotdx.Client.GetIndexBars 固定 times=1、adjust=None（内部写死），键中不再
// 携带这两项，也就不存在 0/1 归一的问题。
func indexBarCacheKey(category uint16, market uint8, code string) string {
	return fmt.Sprintf("indexbars:%d:%d:%s", category, market, code)
}

// GetIndexBars 缓存版指数 K 线查询。
//
// 方案：与 StockKLine 同构——已闭合的指数 K 线不再变化，未闭合的最新一根
// 存实时槽并按 klineLiveRefresh 刷新，已闭合根按「时间戳分数 + 单根
// MessagePack 成员」存 `ZSet`，查询时由基座合并两者、按 offset 窗口返回。
//
// 与原接口的差异只在返回值形状：`gotdx.Client.GetIndexBars` 返回
// `*proto.GetIndexBarsReply`（Count + List），这里直接返回 `[]proto.IndexBar`，
// 与 StockKLine 返回 `[]proto.SecurityBar` 的口径一致，语义等价于读
// `reply.List`（Count 由 len 体现）。除此之外参数含义完全一致：category 为
// `types.KLINE_TYPE_*`，market 为市场代码，code 为指数代码，start 为起始
// 偏移（0 为最新端、增大向更早历史），count 为请求条数；命中缓存时不触碰
// 行情源。
//
// 已知边界：`proto.FixKlineTimeIndexBar` 会把当天下午首根 13:00 的分钟 K 线
// 改写为 11:30，该根与上午 11:30 同分；同一分数在 `ZSet` 中只保留一个成员，
// 因此这种当天数据下缓存比源端少一根。这是协议层既有修正带来的结果，缓存
// 按时间分数排序必须沿用同一时间，故不额外区分。
func (c *CacheClient) GetIndexBars(category uint16, market uint8, code string,
	start uint16, count uint16) ([]proto.IndexBar, error) {
	// 空窗口不触网：与原接口对 count=0 的返回（空列表、无错误）一致。
	if count == 0 {
		return []proto.IndexBar{}, nil
	}
	p := c.newPositionPager(
		indexBarCacheKey(category, market, code),
		// fetch：按原窗口回源（裸调用，熔断与统计由 pager 统一包装）。
		func(ctx context.Context, start, count int) ([]proto.IndexBar, error) {
			reply, err := c.cli.GetIndexBars(category, market, code, uint16(start), uint16(count))
			if err != nil {
				return nil, err
			}
			if reply == nil {
				return nil, nil
			}
			return reply.List, nil
		},
		// head：取最新端两根，最新根进槽、次新根进 `ZSet`。
		func(ctx context.Context) ([]proto.IndexBar, error) {
			reply, err := c.cli.GetIndexBars(category, market, code, 0, klineLiveFetch)
			if err != nil {
				return nil, err
			}
			if reply == nil {
				return nil, nil
			}
			return reply.List, nil
		},
		int(klineLiveFetch),
		// score：越新分数越大（Unix 毫秒）。
		func(b proto.IndexBar) float64 { return float64(b.DateTime.UnixMilli()) },
		klineLiveRefresh,
	)
	ctx := context.Background()
	items, err := p.query(ctx, int(start), int(count))
	if err != nil {
		return nil, err
	}
	return items, nil
}

// exKLine2CacheKey 返回扩展市场 K 线第二变体能力的语义键：按市场类别、
// 代码、周期类别与周期倍数隔离。扩展协议没有复权维度，times 为零按 1
// 归一，与构造器的默认处理保持一致，避免 0/1 分裂成两个键。
func exKLine2CacheKey(category uint8, code string, period, times uint16) string {
	if times == 0 {
		times = 1
	}
	return fmt.Sprintf("exkline2:%d:%s:%d:%d", category, code, period, times)
}

// ExGetKLine2 缓存版扩展市场 K 线查询（协议 `0x2489` 第二变体）。
//
// 方案：与 StockKLine、GetIndexBars 同构——已闭合的根按「时间戳分数 + 单根
// MessagePack 成员」存 `ZSet`，未闭合的最新一根存实时槽并按 klineLiveRefresh
// 刷新，查询时由基座合并两段后按 offset 窗口返回（start 为 offset、count
// 为 limit、start=0 为最新端）。
//
// 与原接口的差异只有返回值形状：`gotdx.Client.ExGetKLine2` 返回
// `*proto.ExGetKLine2Reply`（Category/Name/Period/Times/Start/Count + List），
// 这里直接返回 `[]proto.ExKLineItem`，语义等价于读 `reply.List`（Count 由
// len 体现，Category/Name 等回显字段不返回）。参数含义完全一致，只是偏移
// 与条数按协议宽度用 `uint32`，pager 内部统一按 `int` 承接。
//
// 适用前提：`period` 解码出的时间单调递增（日线、分钟线等周期均满足）。
// 若某个周期的时间戳不单调，位置分页的前提不成立，应直接调用原接口。
//
// 取舍：`ExKLineItem` 的 PreClose/RisePrice/RiseRate 由协议解析时按「上一根
// 收盘」滚动计算，命中缓存后返还的是写入时的快照。对已闭合根而言前一根
// 收盘固定，两者恒等，故不做重算。
func (c *CacheClient) ExGetKLine2(category uint8, code string, period uint16,
	start uint32, count uint32, times uint16) ([]proto.ExKLineItem, error) {
	// count 为 0 时协议构造器只多取一根、随后又剔除，等价于空窗口：
	// 不触网，与原始接口的返回（空列表、无错误）一致。
	if count == 0 {
		return []proto.ExKLineItem{}, nil
	}
	p := c.newPositionPager(
		exKLine2CacheKey(category, code, period, times),
		// fetch：按原窗口回源（裸调用，熔断与统计由 pager 统一包装）。
		// 经由 Client.ExKLine2 而非 Client.ExGetKLine2：后者直接 execute 走
		// 主连接，扩展站客户端（只配 WithExTCPAddress）会报 no available
		// connection；ExKLine2 内部先用 exQuotationClient 完成扩展站连接。
		func(ctx context.Context, start, count int) ([]proto.ExKLineItem, error) {
			reply, err := c.cli.ExGetKLine2(category, code, period, uint32(start), uint32(count), times)
			if err != nil {
				return nil, err
			}
			if reply == nil {
				return nil, nil
			}
			return reply.List, nil
		},
		// head：取最新端两根，最新根进槽、次新根进 `ZSet`。
		func(ctx context.Context) ([]proto.ExKLineItem, error) {
			reply, err := c.cli.ExGetKLine2(category, code, period, 0, klineLiveFetch, times)
			if err != nil {
				return nil, err
			}
			if reply == nil {
				return nil, nil
			}
			return reply.List, nil
		},
		int(klineLiveFetch),
		// score：越新分数越大（Unix 毫秒）。
		func(b proto.ExKLineItem) float64 { return float64(b.DateTime.UnixMilli()) },
		klineLiveRefresh,
	)
	ctx := context.Background()
	items, err := p.query(ctx, int(start), int(count))
	if err != nil {
		return nil, err
	}
	return items, nil
}

// macBarsCacheKey 返回 MAC 证券 K 线能力的语义键：按市场、代码、周期类别、
// 周期倍数与复权方式隔离。times 为零按 1 归一，与协议构造器的默认处理一致。
func macBarsCacheKey(market uint8, code string, period, times, adjust uint16) string {
	if times == 0 {
		times = 1
	}
	return fmt.Sprintf("macbars:%d:%s:%d:%d:%d", market, code, period, times, adjust)
}

// MACSymbolBars 缓存版 MAC 证券 K 线查询。
//
// 方案：与 StockKLine、GetIndexBars、ExGetKLine2 同构——已闭合根按「时间戳
// 分数 + 单根 MessagePack 成员」存 `ZSet`，未闭合的最新一根存实时槽并按
// klineLiveRefresh 刷新，查询时由基座合并两段后按 offset 窗口返回（start 为
// offset、count 为 limit、start=0 为最新端）。参数与返回值与
// `gotdx.Client.MACSymbolBars` 完全一致：market/code/period/times/start/
// count/adjust，返回 `[]proto.MACSymbolBar`。
//
// 回源仍走原方法，因此分页拼接、换手率按流通股本回填
// （applyMACSymbolBarTurnover）与期货 K 线时间换算（resolveMACFuturesDateTimes）
// 都在回源时完成；命中缓存直接返回写入时的快照，不重算换手率。
//
// 已知边界（继承原接口）：`MACSymbolBars` 的多页循环以「不足一页即停」判断，
// 而协议每次多取一根又被剔除，单页实际返回 `DefaultMACSymbolBarsPage-1` 条，
// 因此 count 超过该值时原方法也只返回约一页的量；缓存对此保持同样的返回，
// 且因条数不足 count 而每次都会回源（不做额外补偿）。
func (c *CacheClient) MACSymbolBars(market uint8, code string, period uint16,
	times uint16, start uint32, count uint32, adjust uint16) ([]proto.MACSymbolBar, error) {
	// 空窗口不触网：原方法在 count=0 时同样直接返回空列表。
	if count == 0 {
		return []proto.MACSymbolBar{}, nil
	}
	p := c.newPositionPager(
		macBarsCacheKey(market, code, period, times, adjust),
		// fetch：按原窗口回源（裸调用，熔断与统计由 pager 统一包装）。
		func(ctx context.Context, start, count int) ([]proto.MACSymbolBar, error) {
			return c.cli.MACSymbolBars(market, code, period, times, uint32(start), uint32(count), adjust)
		},
		// head：取最新端两根，最新根进槽、次新根进 `ZSet`。
		func(ctx context.Context) ([]proto.MACSymbolBar, error) {
			return c.cli.MACSymbolBars(market, code, period, times, 0, klineLiveFetch, adjust)
		},
		int(klineLiveFetch),
		// score：越新分数越大（Unix 毫秒）。
		func(b proto.MACSymbolBar) float64 { return float64(b.DateTime.UnixMilli()) },
		klineLiveRefresh,
	)
	ctx := context.Background()
	items, err := p.query(ctx, int(start), int(count))
	if err != nil {
		return nil, err
	}
	return items, nil
}

// positionPager 为「start 为 offset、count 为 limit、index 0 为最新端」
// 型接口的通用缓存引擎，由 newPositionPager 工厂构造。
//
// 字段全部是构造时注入的回源配置，pager 不感知具体业务类型，只按
// score 的单调性维护「实时槽 + `ZSet` 连续前缀」两段式缓存。
type positionPager[T any] struct {
	c   *CacheClient
	key string
	// fetch 按原窗口回源；start/count 统一为 int，各业务方法按自身协议的
	// 宽度（uint16/uint32）自行收窄，pager 不绑定具体接口的参数宽度。
	fetch func(ctx context.Context, start, count int) ([]T, error)
	head  func(ctx context.Context) ([]T, error)
	headN int
	score func(T) float64
	fresh time.Duration
}

// newPositionPager 是位置分页缓存的工厂方法：把一个「最新端在 index 0」
// 型接口的 key 与四个回源/排序配置组装成可复用的 pager。
//
// 接入新方法只需调用它再 p.query(ctx, start, count)，不需要重复实现
// 实时槽刷新、连续前缀判据、singleflight、分布式锁与降级兜底。
func (c *CacheClient) newPositionPager[T any](key string,
	fetch func(ctx context.Context, start, count int) ([]T, error),
	head func(ctx context.Context) ([]T, error),
	headN int,
	score func(T) float64,
	fresh time.Duration) *positionPager[T] {
	if headN < 1 {
		headN = 1
	}
	return &positionPager[T]{
		c: c, key: key,
		fetch: fetch, head: head,
		headN: headN, score: score, fresh: fresh,
	}
}

// query 返回窗口 [start, start+count)，从早到晚排列。
//
// 实时槽的刷新分两种触发条件：
//
//   - start=0：窗口含未闭合的最新一条，必须保证它是新鲜数据，超过 fresh
//     就回源刷新；
//   - start>0：窗口不含最新一条，不按新鲜度刷新（值旧不影响返回内容），
//     但**槽必须存在**：`ZSet` 只存已闭合条目、最新一条在槽里，槽一旦
//     过期，位置基准就整体前移一位，深页窗口会静默返回错位数据。因此
//     槽缺失时补一次，恢复位置完整性。
//
// 流程：按上述条件刷新实时槽 → 读合并窗口（命中的判据是「条数正好等于
// count」，只判无错会把「深度不足」当成命中并静默返回短窗口）→ 未命中或
// 深度不足时走 ensure（原窗口回源，必要时并入缓存）。
func (p *positionPager[T]) query(ctx context.Context, start, count int) ([]T, error) {
	if count == 0 {
		return make([]T, 0), nil
	}
	if start == 0 {
		p.refreshLive(ctx)
	} else if !p.liveExists(ctx) {
		p.refreshLive(ctx)
	}

	got, qerr := p.c.tsQuery[T](ctx, p.key, start, count, 0, nil)
	switch {
	case qerr == nil && len(got) == count:
		p.c.stats.hits.Add(1)
		tsReverse(got)
		return got, nil
	case errors.Is(qerr, errTsEmpty):
		p.c.stats.emptyHits.Add(1)
		return nil, nil
	case errors.Is(qerr, ErrSourceFailed):
		p.c.stats.emptyHits.Add(1)
		return nil, ErrSourceFailed
	}
	items, err := p.ensure(ctx, start, count)
	if err != nil {
		return nil, err
	}
	tsReverse(items)
	return items, nil
}

// refreshLive 维护实时槽的新鲜度与「最新端连续」不变量：槽内数据在
// fresh 内更新过则直接返回，否则回源 head 并写入缓存。
//
// 刷新走同进程 `singleflight`（键与 ensure 区分），避免实时槽过期的瞬间
// 多个并发请求各自回源一次；合并后再做一次新鲜度确认，等锁期间已被别处
// 刷新则直接跳过。回源失败或结果为空时静默返回：刷新失败不影响后续查询
// （查询侧仍会走 ensure 或直连降级），也不写空标记，避免一次刷新失败
// 把窗口请求整体打成失败态。
func (p *positionPager[T]) refreshLive(ctx context.Context) {
	if p.liveFresh(ctx) {
		return
	}
	_, _, _ = p.c.group.Do(p.c.tsKeyBase(p.key)+":refresh", func() (any, error) {
		// 二次确认：等待期间可能已有其他协程完成刷新。
		if p.liveFresh(ctx) {
			return nil, nil
		}
		items, err := p.fetchHeadGuarded(ctx)
		if err != nil || len(items) == 0 {
			return nil, nil
		}
		// 源端可能多返回更旧的数据，只保留最新 headN 条，避免把缓存深度
		// 拉歪成比用户翻页步长多出 1~2 条。
		if len(items) > p.headN {
			items = items[len(items)-p.headN:]
		}
		_ = p.store(ctx, items)
		return nil, nil
	})
}

// liveFresh 判断实时槽是否仍在新鲜度窗口内（读取失败一律视为不新鲜，
// 由调用方决定是否跳过刷新）。
func (p *positionPager[T]) liveFresh(ctx context.Context) bool {
	raw, err := p.c.rds.Get(ctx, p.c.liveKey(p.key)).Result()
	if err != nil {
		return false
	}
	lv, derr := decodeLiveValue(raw)
	if derr != nil {
		return false
	}
	return time.Since(time.Unix(0, lv.TickAt)) <= p.fresh
}

// liveExists 判断实时槽是否存在。位置分页的基准是「槽 + `ZSet`」，槽
// 缺失会让整个位置基准前移一位，深页窗口因此错位，所以 start>0 的查询
// 必须先确认槽在。读取失败按「不存在」处理，交给 refreshLive 尝试补写。
func (p *positionPager[T]) liveExists(ctx context.Context) bool {
	n, err := p.c.rds.Exists(ctx, p.c.liveKey(p.key)).Result()
	return err == nil && n > 0
}

// ensure 为未命中与深度不足时的回源入口，流程对齐基座 tsGet：
//
//   - 先进 `singleflight` 合并同进程并发，回调内抢分布式锁并二次确认；
//   - 一律按**原窗口** (start,count) 回源，不预先拉取 (0, start+count)：
//     深 offset 也只发一次正常条数的请求，不会出现注定失败的超大请求
//     （源端对过大的整段请求会返回 invalid kline item 并污染熔断计数）；
//   - 回源结果只在「缓存已覆盖窗口起点之前」（depth >= start）时才并入
//     缓存，因此写入始终是从最新端起的连续前缀；跳页（depth < start）
//     只直通返回、不写缓存，避免在 `ZSet` 中间留下洞造成位置错位；
//   - 抢锁失败、锁机制故障（如 `Redis` 不可用）时同样直连原窗口兜底，
//     保证行情源可用时不因缓存故障中断。
//
// 已知取舍：请求窗口超出标的实际历史深度（历史尽头）时，回源结果本身
// 就不足 `count`，每次调用都会再回源一次；这类请求罕见且不影响正确性，
// 故不额外引入「深度到头」标记。
func (p *positionPager[T]) ensure(ctx context.Context, start, count int) ([]T, error) {
	v, err, _ := p.c.group.Do(p.c.tsKeyBase(p.key), func() (any, error) {
		var out []T
		lockErr := p.c.lock(ctx, p.c.tsLockKey(p.key), func() error {
			// 二次确认：持锁期间可能已有其他进程补齐数据或写入空标记。
			got, qerr := p.c.tsQuery[T](ctx, p.key, start, count, 0, nil)
			switch {
			case qerr == nil && len(got) == count:
				p.c.stats.hits.Add(1)
				out = got
				return nil
			case errors.Is(qerr, errTsEmpty):
				p.c.stats.emptyHits.Add(1)
				return nil
			case errors.Is(qerr, ErrSourceFailed):
				p.c.stats.emptyHits.Add(1)
				return qerr
			}

			// 按原窗口回源：深 offset 也只是一次正常条数的请求。
			items, ferr := p.fetchWindowGuarded(ctx, start, count)
			if ferr != nil {
				// 熔断拒绝不写失败标记（行情源暂不可用而非确认失败）。
				if !errors.Is(ferr, ErrCircuitOpen) {
					_ = p.c.tsMarkEmpty(ctx, p.key, true)
				}
				return fmt.Errorf("%w: %v", errFetchFailed, ferr)
			}
			if len(items) == 0 {
				_ = p.c.tsMarkEmpty(ctx, p.key, false)
				return nil
			}
			// 用于写入的视图可能比源端返回少：源端会多返回 1~2 条更旧的
			// 数据（协议多取一条推算昨收），写入时只保留窗口需要的 count
			// 条，避免把缓存深度拉歪；返回给调用方的仍是源端原样，与未加
			// 缓存一致。
			storable := items
			if len(storable) > count {
				storable = storable[len(storable)-count:]
			}
			// 缓存已覆盖窗口起点之前才并入，保证 `ZSet` 始终是从最新端起
			// 的连续前缀；跳页时直通不写，避免中间留洞。
			if depth, derr := p.depth(ctx); derr == nil && depth >= start {
				if start == 0 {
					// start=0 的窗口含未闭合的最新一条：拆成实时槽 + 已闭合根。
					_ = p.store(ctx, storable)
				} else {
					// start>0 的窗口不含最新一条，全部是已闭合数据。
					_ = p.storeClosed(ctx, storable)
				}
			}
			tsReverse(items)
			out = items
			return nil
		})

		switch {
		case lockErr == nil:
			return out, nil
		case errors.Is(lockErr, errLockNotAcquired):
			// 抢锁失败：直连原窗口兜底，不写缓存。
			items, ferr := p.fetchWindowGuarded(ctx, start, count)
			if ferr != nil {
				return nil, ferr
			}
			tsReverse(items)
			return items, nil
		case errors.Is(lockErr, errFetchFailed):
			// 锁内回源已失败（含熔断拒绝）：已写失败标记，透传不再重复回源。
			return nil, lockErr
		default:
			// 锁机制故障（如 `Redis` 故障）：直连原窗口兜底，不写缓存。
			items, ferr := p.fetchWindowGuarded(ctx, start, count)
			if ferr != nil {
				return nil, ferr
			}
			tsReverse(items)
			return items, nil
		}
	})
	if err != nil {
		return nil, err
	}
	return v.([]T), nil
}

// fetchWindowGuarded 按原窗口回源，统一经过熔断保护与回源统计。
func (p *positionPager[T]) fetchWindowGuarded(ctx context.Context, start, count int) ([]T, error) {
	var items []T
	gerr := p.c.fetchGuarded(ctx, func(ctx context.Context) error {
		var e error
		items, e = p.fetch(ctx, start, count)
		return e
	})
	if gerr != nil {
		return nil, gerr
	}
	return items, nil
}

// fetchHeadGuarded 取最新端数据，统一经过熔断保护与回源统计。
func (p *positionPager[T]) fetchHeadGuarded(ctx context.Context) ([]T, error) {
	var items []T
	gerr := p.c.fetchGuarded(ctx, func(ctx context.Context) error {
		var e error
		items, e = p.head(ctx)
		return e
	})
	if gerr != nil {
		return nil, gerr
	}
	return items, nil
}

// depth 返回缓存当前「从最新端起的连续前缀深度」：`ZSet` 条数加上存在
// 的实时槽（未闭合最新一条）。跳页写入前用它判断能否安全并入。
func (p *positionPager[T]) depth(ctx context.Context) (int, error) {
	pipe := p.c.rds.Pipeline()
	card := pipe.ZCard(ctx, p.c.tsKeyBase(p.key))
	live := pipe.Exists(ctx, p.c.liveKey(p.key))
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	depth := int(card.Val())
	if live.Val() > 0 {
		depth++
	}
	return depth, nil
}

// store 把一批从旧到新的数据拆成「已闭合条目 + 未闭合最新一条」写入：
// 除最后一条外全部按分数幂等写入 `ZSet`，最后一条覆盖写实时槽。
//
// 调用方须保证这批数据是从最新端起的连续前缀（start=0 的窗口，或紧接
// 在已缓存深度之后的窗口），否则 `ZSet` 中间会留洞。
func (p *positionPager[T]) store(ctx context.Context, items []T) error {
	return p.c.storeWindow(ctx, p.key, items, p.score)
}

// storeClosed 把一批已闭合条目按分数幂等写入 `ZSet`：同分先删后加，
// 替换历史修正，也保证重复并入同一窗口时不堆积重复成员。
func (p *positionPager[T]) storeClosed(ctx context.Context, items []T) error {
	if len(items) == 0 {
		return nil
	}
	scores := make([]float64, len(items))
	for i := range items {
		scores[i] = p.score(items[i])
	}
	return p.c.positionWriteClosed(ctx, p.key, items, scores)
}

// storeWindow 把一批从旧到新的条目拆成「已闭合 + 未闭合最新一条」写入，
// 分数由调用方给出的 score 计算。这是位置分页型数据的通用写入路径。
func (c *CacheClient) storeWindow[T any](ctx context.Context, zkey string, items []T, score func(T) float64) error {
	if len(items) == 0 {
		return nil
	}
	if len(items) > 1 {
		closed := items[:len(items)-1]
		scores := make([]float64, len(closed))
		for i := range closed {
			scores[i] = score(closed[i])
		}
		if err := c.positionWriteClosed(ctx, zkey, closed, scores); err != nil {
			return err
		}
	}
	last := items[len(items)-1]
	_, err := c.positionWriteLive(ctx, zkey, last, score(last))
	return err
}

// positionWriteClosed 把一批已闭合条目按分数幂等写入 `ZSet`。
func (c *CacheClient) positionWriteClosed[T any](ctx context.Context, zkey string, items []T, scores []float64) error {
	if len(items) == 0 {
		return nil
	}
	return c.tsPushBatch(ctx, zkey, items, scores, false, true)
}

// positionWriteLive 用 positionLiveScript 原子更新实时槽：同分覆盖、跨分
// 时把前一条幂等流转进 `ZSet`，并续写 `TTL`。返回是否更新生效（现存槽
// 分数更新时为假，表示新值已过期或乱序）。
func (c *CacheClient) positionWriteLive[T any](ctx context.Context, zkey string, item T, score float64) (bool, error) {
	data, err := msgpackEncode(item)
	if err != nil {
		return false, err
	}
	val := encodeLiveValue(score, time.Now().UnixNano(), data)
	// `TTL` 不足 1 秒时按 1 秒处理，避免 `Redis` SET EX 0 立即过期。
	ttlSec := int64(c.opt.LiveTTL / time.Second)
	if ttlSec < 1 {
		ttlSec = 1
	}
	n, err := positionLiveScript.Run(ctx, c.rds,
		[]string{c.liveKey(zkey), c.tsKeyBase(zkey)},
		val, strconv.FormatFloat(score, 'f', -1, 64), ttlSec).Int()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// klineStore 把一批从旧到新的 K 线写入缓存：除最后一根外全部按分数幂等
// 写入 `ZSet`，最后一根覆盖写实时槽（同分覆盖，保证盘中演化能刷新到最新
// 数值）。分数取 K 线的时间（Unix 毫秒）。
//
// 保留此方法作为 K 线的便捷写入入口（与 storeWindow 等价，只是分数来源
// 固定为 DateTime），便于构造缓存前置状态与测试使用。
func (c *CacheClient) klineStore(ctx context.Context, zkey string, bars []proto.SecurityBar) error {
	return c.storeWindow(ctx, zkey, bars, func(b proto.SecurityBar) float64 {
		return float64(b.DateTime.UnixMilli())
	})
}

// klineUpsertLive 用 positionLiveScript 原子更新 K 线的实时槽：同分覆盖、
// 跨分时把前一根幂等流转进 `ZSet`，并续写 `TTL`。
func (c *CacheClient) klineUpsertLive(ctx context.Context, zkey string, bar proto.SecurityBar) (bool, error) {
	return c.positionWriteLive(ctx, zkey, bar, float64(bar.DateTime.UnixMilli()))
}

// 文件 ts.go 实现时间序列能力的存储机制：`ZSet` 存闭合历史（永久保存，
// 不自动淘汰），`String` 实时槽存未闭合的最新一根，查询时合并排序返回
// 窗口数据；配套空标记、tick 刷新与回源补齐。
//
// 分数约定：统一为 Unix 毫秒（时间戳换算），同分至多一根由实时槽
// 唯一性保证；逐笔/成交族需要同分多笔共存时以追加写入绕过该约束。
// 单根数据统一以 MessagePack 编码（比 `gob` 更快更小），再 base64 为
// 文本便于 `Lua` 脚本原样搬移。

package cache

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// liveValue 为实时槽解析结果：分数用于与 `ZSet` 合并排序，更新时刻
// 用于判断是否需要 tick 刷新，数据为单根应答 MessagePack 编码后的字节。
type liveValue struct {
	Score  float64
	TickAt int64 // Unix 纳秒
	Data   []byte
}

// encodeLiveValue 按「分数|更新时刻|base64 数据」拼接实时槽值，分数放
// 在首位便于 `Lua` 脚本按分隔符解析比较。
func encodeLiveValue(score float64, tickAt int64, data []byte) string {
	return strconv.FormatFloat(score, 'f', -1, 64) + "|" +
		strconv.FormatInt(tickAt, 10) + "|" +
		base64.StdEncoding.EncodeToString(data)
}

// decodeLiveValue 解析实时槽值为分数、更新时刻与原始数据。
func decodeLiveValue(s string) (*liveValue, error) {
	parts := strings.SplitN(s, "|", 3)
	if len(parts) != 3 {
		return nil, errors.New("invalid live value")
	}
	score, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return nil, err
	}
	tickAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	data, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	return &liveValue{Score: score, TickAt: tickAt, Data: data}, nil
}

// encodeMember 把单根应答 MessagePack 编码后再 base64 编码为 `ZSet`
// 成员文本；实时槽直接复用该文本，`Lua` 脚本可原样搬移，查询侧按同样
// 顺序解码。
func encodeMember[T any](item T) (string, error) {
	data, err := msgpackEncode(item)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// decodeMember 把 `ZSet` 成员文本解码还原为单根应答。
func decodeMember[T any](member string) (T, error) {
	data, err := base64.StdEncoding.DecodeString(member)
	if err != nil {
		var zero T
		return zero, err
	}
	return msgpackDecode[T](data)
}

// tsKeyBase 返回时间序列语义键对应的实际 `ZSet` 键（带 :ts: 类型段，
// 避免与普通键值键冲突）。
func (c *CacheClient) tsKeyBase(zkey string) string {
	return c.opt.KeyPrefix + ":ts:" + zkey
}

// liveKey 返回时间序列语义键对应的实时槽键。
func (c *CacheClient) liveKey(zkey string) string {
	return c.tsKeyBase(zkey) + ":live"
}

// tsEmptyKey 返回时间序列语义键对应的空标记键。
func (c *CacheClient) tsEmptyKey(zkey string) string {
	return c.tsKeyBase(zkey) + ":empty"
}

// tsLockKey 返回时间序列语义键对应的分布式锁键。
func (c *CacheClient) tsLockKey(zkey string) string {
	return c.tsKeyBase(zkey) + ":lock"
}

// tsTick 用单条 `Lua` 原子完成「同根覆盖 / 前一根闭合流转 / 续写 TTL」：
// 现存实时槽分数不小于新分数时忽略（防乱序），否则把现存前一根追加进
// `ZSet` 并以新值覆盖实时槽续写 `TTL`。返回是否写入生效。
func (c *CacheClient) tsTick[T any](ctx context.Context, zkey string, item T, score float64, ttl time.Duration) (bool, error) {
	data, err := msgpackEncode(item)
	if err != nil {
		return false, err
	}
	val := encodeLiveValue(score, time.Now().UnixNano(), data)
	// TTL 不足 1 秒时按 1 秒处理，避免 `Redis` SET EX 0 立即过期。
	ttlSec := int64(ttl / time.Second)
	if ttlSec < 1 {
		ttlSec = 1
	}
	n, err := liveScript.Run(ctx, c.rds,
		[]string{c.liveKey(zkey), c.tsKeyBase(zkey)},
		val, strconv.FormatFloat(score, 'f', -1, 64), ttlSec).Int()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// tsPushBatch 把整批回源数据写入缓存：先统一编码全部成员，再以
// pipeline 批量写入 `ZSet`；live 为真时最后一条优先写入实时槽（作为
// 未闭合的最新一根），若现存实时槽分数已更新则把最后一条改入 `ZSet`。
// uniqueScore 为真时每条先删同分数旧成员再追加（覆盖语义），为假时
// 允许同分数多笔共存（逐笔/成交追加语义）。
func (c *CacheClient) tsPushBatch[T any](ctx context.Context, zkey string, batch []T, scores []float64, live, uniqueScore bool) error {
	if len(batch) == 0 {
		return nil
	}
	if len(batch) != len(scores) {
		return errors.New("batch and scores length mismatch")
	}
	zk := c.tsKeyBase(zkey)
	// 最后一条留给实时槽（若 live 为真），其余全部写入 `ZSet`。
	n := len(batch)
	if live {
		n--
	}
	// 先编码全部成员，再批量提交，避免 pipeline 中途失败留下半批。
	entries := make([]redis.Z, 0, n)
	for i := 0; i < n; i++ {
		member, err := encodeMember(batch[i])
		if err != nil {
			return err
		}
		entries = append(entries, redis.Z{Score: scores[i], Member: member})
	}
	if len(entries) > 0 {
		pipe := c.rds.Pipeline()
		for _, e := range entries {
			if uniqueScore {
				pipe.Eval(ctx, rewriteScriptSrc, []string{zk},
					strconv.FormatFloat(e.Score, 'f', -1, 64), e.Member)
			} else {
				pipe.ZAdd(ctx, zk, e)
			}
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	if live {
		last := batch[len(batch)-1]
		lastScore := scores[len(scores)-1]
		ok, err := c.tsTick(ctx, zkey, last, lastScore, c.opt.LiveTTL)
		if err != nil {
			return err
		}
		if !ok {
			// 现存实时槽分数已更新：最后一条已闭合，改入 `ZSet`。
			member, err := encodeMember(last)
			if err != nil {
				return err
			}
			if uniqueScore {
				return rewriteScript.Run(ctx, c.rds, []string{zk},
					strconv.FormatFloat(lastScore, 'f', -1, 64), member).Err()
			}
			return c.rds.ZAdd(ctx, zk, redis.Z{Score: lastScore, Member: member}).Err()
		}
	}
	return nil
}

// tsQuery 按窗口返回合并后的时间序列数据：用一条 pipeline 同时完成
// 空标记检查、实时槽读取与 `ZSet` 最新段读取（单次往返）；实时槽超过
// refresh 未更新时用 latest 回调刷新一次。
//
// 返回错误：命中失败空标记返回 ErrSourceFailed，命中空结果标记返回
// errTsEmpty，键不存在或窗口起始越界返回 errTsMiss。
func (c *CacheClient) tsQuery[T any](ctx context.Context, zkey string, start, count int,
	refresh time.Duration,
	latest func(context.Context) (T, float64, error)) ([]T, error) {
	zk := c.tsKeyBase(zkey)
	lk := c.liveKey(zkey)
	limit := int64(start + count)
	if limit < 0 {
		limit = 0
	}

	// 单次往返完成全部读取：空标记、实时槽、`ZSet` 最新段。
	pipe := c.rds.Pipeline()
	emptyCmd := pipe.Get(ctx, c.tsEmptyKey(zkey))
	liveCmd := pipe.Get(ctx, lk)
	zrangeCmd := pipe.ZRevRangeWithScores(ctx, zk, 0, limit)
	_, _ = pipe.Exec(ctx)

	// 空标记检查：命中即不再回源。
	if emptyCmd.Err() == nil {
		s, _ := emptyCmd.Result()
		if s == emptyFlagFail {
			return nil, ErrSourceFailed
		}
		return nil, errTsEmpty
	}

	// 读取实时槽。
	var liveItem *liveValue
	if liveCmd.Err() == nil {
		if s, err := liveCmd.Result(); err == nil {
			if lv, derr := decodeLiveValue(s); derr == nil {
				liveItem = lv
			}
		}
	}

	// 实时槽超过 refresh 未更新时用 latest 刷新一次；刷新同样受熔断
	// 保护（行情源故障时不因刷新反复重试），失败不影响返回。
	if liveItem != nil && refresh > 0 && latest != nil {
		if time.Since(time.Unix(0, liveItem.TickAt)) > refresh {
			var item T
			var score float64
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				item, score, e = latest(ctx)
				return e
			})
			if gerr == nil {
				if ok, terr := c.tsTick(ctx, zkey, item, score, c.opt.LiveTTL); ok && terr == nil {
					if s, err := c.rds.Get(ctx, lk).Result(); err == nil {
						if lv, derr := decodeLiveValue(s); derr == nil {
							liveItem = lv
						}
					}
				}
			}
		}
	}

	// 读取 `ZSet` 最新段（窗口末端之前，避免全量读取）。
	if err := zrangeCmd.Err(); err != nil {
		return nil, err
	}
	zs := zrangeCmd.Val()

	// 合并实时槽与闭合历史，按分数降序（时间从新到旧）排序。
	type entry struct {
		score float64
		data  []byte
	}
	all := make([]entry, 0, len(zs)+1)
	for _, z := range zs {
		member, ok := z.Member.(string)
		if !ok {
			continue
		}
		data, derr := base64.StdEncoding.DecodeString(member)
		if derr != nil {
			continue
		}
		all = append(all, entry{z.Score, data})
	}
	if liveItem != nil {
		all = append(all, entry{liveItem.Score, liveItem.Data})
	}
	if len(all) == 0 {
		return nil, errTsMiss
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })

	// 按窗口切片返回；start 为负时按 0 处理，越界视为未命中。
	if start < 0 {
		start = 0
	}
	if start >= len(all) {
		return nil, errTsMiss
	}
	end := start + count
	if end > len(all) {
		end = len(all)
	}
	out := make([]T, 0, end-start)
	for _, e := range all[start:end] {
		v, gerr := msgpackDecode[T](e.data)
		if gerr != nil {
			return nil, gerr
		}
		out = append(out, v)
	}
	return out, nil
}

// tsGet 为时间序列能力的统一查询入口，流程与 kvGet 对应：
//
//   - 空标记命中：失败标记返回 ErrSourceFailed，空结果标记返回空
//     切片（len 为 0，无错误）。
//   - 缓存命中：直接返回合并窗口数据。
//   - 未命中：先进 `singleflight` 合并同进程并发，回调内抢分布式锁；
//     持锁方二次确认后调用 fetch 回源并整批写入，写入后直接按窗口
//     返回回源数据（从新到旧）；回源确认无数据写空结果标记，回源
//     失败写失败标记。
//   - 抢锁失败或锁不可用（如 `Redis` 故障）：直连 fetch 回源（不写
//     缓存），结果反转后返回，保证行情源可用时不因缓存故障中断。
func (c *CacheClient) tsGet[T any](ctx context.Context, zkey string, start, count int,
	live, uniqueScore bool, refresh time.Duration,
	latest func(context.Context) (T, float64, error),
	fetch func(context.Context) ([]T, []float64, bool, error)) ([]T, error) {
	items, err := c.tsQuery(ctx, zkey, start, count, refresh, latest)
	if err == nil {
		c.stats.hits.Add(1)
		return items, nil
	}
	if errors.Is(err, errTsEmpty) {
		// 空结果标记命中：按空数据处理。
		c.stats.emptyHits.Add(1)
		return nil, nil
	}
	if errors.Is(err, ErrSourceFailed) {
		c.stats.emptyHits.Add(1)
		return nil, err
	}
	if !errors.Is(err, errTsMiss) {
		// 缓存层故障（如 `Redis` 不可用或数据损坏）：直连回源兜底，
		// 保证行情源可用时不因缓存故障中断。
		var batch []T
		gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
			var e error
			batch, _, _, e = fetch(ctx)
			return e
		})
		if gerr != nil {
			return nil, gerr
		}
		tsReverse(batch)
		return batch, nil
	}
	// 未命中：`singleflight` 合并同进程并发，回调内抢分布式锁并二次确认；
	// 持锁方回源补齐，抢锁失败在回调内直连回源，同进程只打一次行情源。
	v, err, _ := c.group.Do(c.tsKeyBase(zkey), func() (any, error) {
		var out []T
		lockErr := c.lock(ctx, c.tsLockKey(zkey), func() error {
			// 二次确认：持锁期间可能已有其他进程补齐数据或写入空标记。
			if got, qerr := c.tsQuery(ctx, zkey, start, count, refresh, latest); qerr == nil {
				c.stats.hits.Add(1)
				out = got
				return nil
			} else if errors.Is(qerr, errTsEmpty) {
				// 空结果标记命中：按空数据处理，不再回源。
				c.stats.emptyHits.Add(1)
				return nil
			} else if errors.Is(qerr, ErrSourceFailed) {
				c.stats.emptyHits.Add(1)
				return qerr
			}
			var batch []T
			var scores []float64
			var empty bool
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				batch, scores, empty, e = fetch(ctx)
				return e
			})
			if gerr != nil {
				// 熔断拒绝时不写失败标记（行情源暂不可用而非确认失败），
				// 熔断恢复后重新回源；回源失败统一包装为哨兵错误，外层
				// 据此不再重复直连。
				if !errors.Is(gerr, ErrCircuitOpen) {
					_ = c.tsMarkEmpty(ctx, zkey, true)
				}
				return fmt.Errorf("%w: %v", errFetchFailed, gerr)
			}
			if empty {
				_ = c.tsMarkEmpty(ctx, zkey, false)
				return nil
			}
			if werr := c.tsPushBatch(ctx, zkey, batch, scores, live, uniqueScore); werr != nil {
				return werr
			}
			// 回源数据不足窗口时直接返回已有部分（从新到旧），避免
			// 下次查询再次判定未命中而反复回源。
			tsReverse(batch)
			if start >= len(batch) {
				return errTsMiss
			}
			end := start + count
			if end > len(batch) {
				end = len(batch)
			}
			out = batch[start:end]
			return nil
		})
		if lockErr != nil && !errors.Is(lockErr, errLockNotAcquired) && !errors.Is(lockErr, errFetchFailed) {
			// 锁机制故障（如 `Redis` 故障）：直连回源兜底，不写缓存。
			var batch []T
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				batch, _, _, e = fetch(ctx)
				return e
			})
			if gerr != nil {
				return nil, gerr
			}
			tsReverse(batch)
			return batch, nil
		}
		// 抢锁失败：直连回源兜底（不写缓存），结果从旧到新反转后返回。
		if errors.Is(lockErr, errLockNotAcquired) {
			var batch []T
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				batch, _, _, e = fetch(ctx)
				return e
			})
			if gerr != nil {
				return nil, gerr
			}
			tsReverse(batch)
			return batch, nil
		}
		// 锁内回源已失败（含熔断拒绝）：已写失败标记，透传错误，不再
		// 重复打行情源。
		if errors.Is(lockErr, errFetchFailed) {
			return nil, lockErr
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]T), nil
}

// tsMarkEmpty 写入空结果标记：fail 为真表示回源失败（后续直接返回
// 错误），为假表示确认无数据（后续按空数据处理）。标记带短 TTL 防穿透。
func (c *CacheClient) tsMarkEmpty(ctx context.Context, zkey string, fail bool) error {
	flag := emptyFlagOK
	if fail {
		flag = emptyFlagFail
	}
	return c.rds.Set(ctx, c.tsEmptyKey(zkey), flag, c.opt.EmptyTTL).Err()
}

// tsReverse 原地反转切片（回源按时间从旧到新返回，查询需要从新到旧）。
func tsReverse[T any](batch []T) {
	for i, j := 0, len(batch)-1; i < j; i, j = i+1, j-1 {
		batch[i], batch[j] = batch[j], batch[i]
	}
}

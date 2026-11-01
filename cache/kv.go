// 文件 kv.go 实现普通键值能力的缓存机制：整段应答 MessagePack 编码后
// 存入 `Redis` `String`，附带空标记与分布式锁；未命中路径先经
// `singleflight` 合并同进程并发，再抢分布式锁回源，双层防护避免击穿
// 行情源。

package cache

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// kvKey 返回普通键值语义键对应的实际 `Redis` 键（带 :kv: 类型段，避免
// 与时间序列键冲突）。
func (c *CacheClient) kvKey(k string) string {
	return c.opt.KeyPrefix + ":kv:" + k
}

// kvEmptyKey 返回普通键值语义键对应的空标记键。
func (c *CacheClient) kvEmptyKey(k string) string {
	return c.kvKey(k) + ":empty"
}

// kvLockKey 返回普通键值语义键对应的分布式锁键。
func (c *CacheClient) kvLockKey(k string) string {
	return c.kvKey(k) + ":lock"
}

// kvGet 为普通键值能力的统一查询入口，流程如下：
//
//   - 空标记命中：失败标记返回 ErrSourceFailed，空结果标记返回 T 的
//     零值（nil error），两者都不再回源。
//   - 缓存命中：直接解码返回，不触碰回源。
//   - 未命中：先进 `singleflight` 合并同进程并发，回调内抢分布式锁；
//     持锁方二次确认后调用 fetch 回源；fetch 返回的 empty=true 写空
//     结果标记，err 非空写失败标记；其余情况把应答整段写入缓存并续
//     TTL（超体积上限不写）。
//   - 抢锁失败：直连 fetch 回源（不加锁、不写缓存），结果直接返回。
//   - 锁不可用（如 `Redis` 故障）：同样直连 fetch 回源兜底，保证行情
//     源可用时不因缓存故障中断。
func (c *CacheClient) kvGet[T any](ctx context.Context, key string, ttl time.Duration,
	fetch func(context.Context) (T, bool, error)) (T, error) {
	// 单次往返完成空标记与缓存值读取。
	pipe := c.rds.Pipeline()
	emptyCmd := pipe.Get(ctx, c.kvEmptyKey(key))
	valCmd := pipe.Get(ctx, c.kvKey(key))
	_, _ = pipe.Exec(ctx)

	// 空标记检查：命中即不再回源。
	if emptyCmd.Err() == nil {
		c.stats.emptyHits.Add(1)
		s, _ := emptyCmd.Result()
		if s == emptyFlagFail {
			var zero T
			return zero, ErrSourceFailed
		}
		var zero T
		return zero, nil
	}
	// 缓存命中。
	if valCmd.Err() == nil {
		raw, _ := valCmd.Result()
		env, derr := decodeReplyEnvelope([]byte(raw))
		if derr == nil {
			c.stats.hits.Add(1)
			return msgpackDecode[T](env.Data)
		}
	}
	// 未命中：`singleflight` 合并同进程并发，回调内抢分布式锁并二次确认；
	// 持锁方回源写入，抢锁失败在回调内直连回源，同进程只打一次行情源。
	v, err, _ := c.group.Do(c.kvKey(key), func() (any, error) {
		var out T
		lockErr := c.lock(ctx, c.kvLockKey(key), func() error {
			// 二次确认：单次往返重读缓存与空标记，持锁期间可能已有
			// 其他进程写入。
			pipe := c.rds.Pipeline()
			valCmd := pipe.Get(ctx, c.kvKey(key))
			emptyCmd := pipe.Get(ctx, c.kvEmptyKey(key))
			_, _ = pipe.Exec(ctx)
			if valCmd.Err() == nil {
				raw, _ := valCmd.Result()
				if env, derr := decodeReplyEnvelope([]byte(raw)); derr == nil {
					if val, gerr := msgpackDecode[T](env.Data); gerr == nil {
						c.stats.hits.Add(1)
						out = val
						return nil
					}
				}
			}
			if emptyCmd.Err() == nil {
				c.stats.emptyHits.Add(1)
				s, _ := emptyCmd.Result()
				if s == emptyFlagFail {
					return ErrSourceFailed
				}
				return nil
			}
			var fv T
			var fempty bool
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				fv, fempty, e = fetch(ctx)
				return e
			})
			if gerr != nil {
				// 熔断拒绝时不写失败标记（行情源暂不可用而非确认失败），
				// 熔断恢复后重新回源；回源失败统一包装为哨兵错误，外层
				// 据此不再重复直连。
				if !errors.Is(gerr, ErrCircuitOpen) {
					_ = c.rds.Set(ctx, c.kvEmptyKey(key), emptyFlagFail, c.opt.EmptyTTL).Err()
				}
				return fmt.Errorf("%w: %v", errFetchFailed, gerr)
			}
			v, empty := fv, fempty
			if empty {
				_ = c.rds.Set(ctx, c.kvEmptyKey(key), emptyFlagOK, c.opt.EmptyTTL).Err()
				out = v
				return nil
			}
			data, eerr := msgpackEncode(v)
			if eerr == nil {
				env := &replyEnvelope{Version: 1, Data: data}
				if raw, eerr2 := encodeReplyEnvelope(env); eerr2 == nil {
					_ = c.rds.Set(ctx, c.kvKey(key), raw, ttl).Err()
				}
			}
			out = v
			return nil
		})
		if lockErr != nil && !errors.Is(lockErr, errLockNotAcquired) && !errors.Is(lockErr, errFetchFailed) {
			// 锁机制故障（如 `Redis` 故障）：直连回源兜底，不写缓存。
			var fv T
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				fv, _, e = fetch(ctx)
				return e
			})
			if gerr != nil {
				var zero T
				return zero, gerr
			}
			return fv, nil
		}
		// 抢锁失败：直连回源兜底（不加锁、不写缓存），结果直接返回。
		if errors.Is(lockErr, errLockNotAcquired) {
			var fv T
			gerr := c.fetchGuarded(ctx, func(ctx context.Context) error {
				var e error
				fv, _, e = fetch(ctx)
				return e
			})
			if gerr != nil {
				var zero T
				return zero, gerr
			}
			return fv, nil
		}
		// 锁内回源已失败（含熔断拒绝）：已写失败标记，透传错误，不再
		// 重复打行情源。
		if errors.Is(lockErr, errFetchFailed) {
			var zero T
			return zero, lockErr
		}
		return out, nil
	})
	if err != nil {
		var zero T
		return zero, err
	}
	return v.(T), nil
}

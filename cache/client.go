// Package cache 为 gotdx 提供基于 `Redis` 的缓存层，对外方法与
// gotdx.Client 同名同签名（仅首位新增 context.Context），并发安全。
//
// # 架构总述
//
// 缓存层把所有能力按数据形态划分为两类，分别采用两套存储机制：
//
//   - 时间序列能力（K 线、分时、逐笔、历史成交、竞价、指数 K 线、
//     MAC 各族时序、扩展各族时序等）采用「`ZSet` 闭合历史 + `String`
//     实时」双结构存储，通过 `Lua` 脚本原子流转；详见 ts.go 与各能力
//     文件的方案注释。
//   - 普通键值能力（行情快照、证券列表、榜单、服务器信息、公司资料、
//     文件下载等）把整段应答 MessagePack 编码后存入 `Redis` `String`，
//     附带空标记与分布式锁；详见 kv.go。
//
// # 时间序列存储设计
//
//   - 未闭合的「最新一根」存入普通 `String`（键为 `ZSet` 键加 :live 后缀），
//     值为「分数|更新时刻|base64 数据」三段拼接文本，分数放首位便于 `Lua`
//     脚本解析比较；`tick` 到达时由 `Lua` 脚本原子完成「同根覆盖 / 前一根
//     闭合流转 / 续写 TTL」三件事，`TTL` 仅作行情源断流时的兜底，不作为
//     正常淘汰手段。
//   - 已闭合的根按分数存入 `ZSet`，成员为单根应答 MessagePack 编码后
//     再 base64 编码的文本，与 `String` 实时槽的编码一致，`Lua` 脚本
//     可原样搬移；分数由时间戳换算（统一为 Unix 毫秒），同分至多一根，
//     天然支持按分数精确修正。
//   - 历史修正写入由 `Lua` 脚本原子完成：先按分数删除同分数旧成员再
//     ZADD，保证修正覆盖而不是同分数堆积重复。
//   - `ZSet` 为永久保存，不自动淘汰；历史保留多久由外部决定（如需清理
//     只能显式 `DEL` 或 `ZREMRANGEBYSCORE`），实时槽的 `TTL` 仅作
//     断流兜底，不构成正常淘汰手段。
//   - 查询时把 `ZSet` 闭合历史（ZREVRANGE 取最新段）与 `String` 实时槽
//     中未闭合的最新一根合并，按分数降序（时间从新到旧）返回，与请求的
//     start/count 窗口语义保持一致。
//
// 普通键值与时间序列键分别带 :kv: 与 :ts: 类型段，实际 `Redis` 键互不
// 冲突；同类型内部键的命名空间由能力层的语义键保证（每个能力族一个
// 文件，键由能力名 + 操作标签 + 参数构成）。
//
// # 穿透与击穿防护
//
//   - 回源失败或回源结果为空时写入短 TTL 空标记（默认 30 秒），空标记
//     存活期内不再回源，防止恶意穿透打垮行情源。
//   - 击穿防护分两层：同进程内 `singleflight` 把同键并发合并为一次回源；
//     跨进程由 `Redis` 分布式锁合并（SET NX EX，锁超时默认 5 秒防死锁，
//     持锁方回源写入后以 `Lua` 校验 token 解锁），持锁方在锁内二次确认
//     后才回源；抢锁失败者直连回源兜底，保证行情源不被打垮。
package cache

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync/atomic"
	"time"

	"github.com/bensema/gotdx"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/sync/singleflight"
)

// 默认配置项取值。
const (
	// DefaultKeyPrefix 为全部缓存键的共享前缀。
	DefaultKeyPrefix = "gotdx"
	// DefaultEmptyTTL 为空标记的存活时长，期间不再回源。
	DefaultEmptyTTL = 30 * time.Second
	// DefaultLockTimeout 为分布式锁的最长持有时长，超时自动释放防死锁。
	DefaultLockTimeout = 5 * time.Second
	// DefaultLiveTTL 为 `String` 实时槽的兜底存活时长，仅行情源断流时兜底。
	DefaultLiveTTL = time.Hour
	// DefaultCircuitFailThreshold 为回源连续失败多少次后打开熔断。
	DefaultCircuitFailThreshold = 5
	// DefaultCircuitOpenTimeout 为熔断打开后的冷却时长，期满放行试探请求。
	DefaultCircuitOpenTimeout = 10 * time.Second
)

// 空标记取值：区分回源失败与确认无数据两种情形。
const (
	emptyFlagFail = "fail"
	emptyFlagOK   = "ok"
)

// 包内错误定义，供能力层按语义分发。
var (
	// ErrSourceFailed 表示回源失败且已写入失败空标记。
	ErrSourceFailed = errors.New("source failed")
	// errLockNotAcquired 表示分布式锁获取失败。
	errLockNotAcquired = errors.New("lock not acquired")
	// errTsMiss 表示时序缓存未命中（键不存在或窗口数据不足）。
	errTsMiss = errors.New("ts cache miss")
	// errTsEmpty 表示时序缓存命中空标记。
	errTsEmpty = errors.New("ts cache empty")
	// errFetchFailed 表示锁回调内回源已失败（含熔断拒绝）：外层据此
	// 直接透传，不再走直连兜底重复打行情源。
	errFetchFailed = errors.New("fetch failed in lock")
)

// Options 为缓存层行为参数，未设置的项使用对应默认值。`Redis` 连接的
// 创建与配置由外部完成，经 NewCacheClient 传入，不在本包内管理。
type Options struct {
	// KeyPrefix 为全部缓存键的共享前缀，避免多实例键冲突。
	KeyPrefix string
	// EmptyTTL 为空标记存活时长。
	EmptyTTL time.Duration
	// LockTimeout 为分布式锁最长持有时长。
	LockTimeout time.Duration
	// LiveTTL 为 `String` 实时槽兜底存活时长。
	LiveTTL time.Duration
	// CircuitFailThreshold 为回源连续失败多少次后打开熔断。
	CircuitFailThreshold int
	// CircuitOpenTimeout 为熔断打开后的冷却时长，期满放行试探请求。
	CircuitOpenTimeout time.Duration
}

// defaultOptions 返回全部使用默认值的配置。
func defaultOptions() *Options {
	return &Options{
		KeyPrefix:            DefaultKeyPrefix,
		EmptyTTL:             DefaultEmptyTTL,
		LockTimeout:          DefaultLockTimeout,
		LiveTTL:              DefaultLiveTTL,
		CircuitFailThreshold: DefaultCircuitFailThreshold,
		CircuitOpenTimeout:   DefaultCircuitOpenTimeout,
	}
}

// Option 以函数式方式修改 Options 的单个字段。
type Option func(*Options)

// WithKeyPrefix 设置全部缓存键的共享前缀。
func WithKeyPrefix(prefix string) Option {
	return func(o *Options) { o.KeyPrefix = prefix }
}

// WithEmptyTTL 设置空标记存活时长。
func WithEmptyTTL(ttl time.Duration) Option {
	return func(o *Options) { o.EmptyTTL = ttl }
}

// WithLockTimeout 设置分布式锁最长持有时长。
func WithLockTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.LockTimeout = timeout }
}

// WithLiveTTL 设置 `String` 实时槽兜底存活时长。
func WithLiveTTL(ttl time.Duration) Option {
	return func(o *Options) { o.LiveTTL = ttl }
}

// WithCircuitFailThreshold 设置回源连续失败多少次后打开熔断。
func WithCircuitFailThreshold(n int) Option {
	return func(o *Options) { o.CircuitFailThreshold = n }
}

// WithCircuitOpenTimeout 设置熔断打开后的冷却时长，期满放行试探请求。
func WithCircuitOpenTimeout(timeout time.Duration) Option {
	return func(o *Options) { o.CircuitOpenTimeout = timeout }
}

// CacheClient 为缓存层的对外入口：持有外部传入的 `Redis` 客户端与行情源
// `gotdx.Client`（能力层回源直接调用它）、同进程 `singleflight` 合并器、
// 回源熔断器与运行时统计。所有方法并发安全。
type CacheClient struct {
	opt     *Options
	rds     *redis.Client
	cli     *gotdx.Client
	group   singleflight.Group
	breaker *circuitBreaker
	stats   clientStats
}

// clientStats 为运行时统计的原子计数集合。
type clientStats struct {
	hits           atomic.Uint64 // 缓存命中次数（数据来自缓存）
	emptyHits      atomic.Uint64 // 空标记命中次数（含失败标记与空结果标记）
	fetches        atomic.Uint64 // 实际执行回源次数（成功与失败都计入）
	sourceErrors   atomic.Uint64 // 回源失败次数
	circuitRejects atomic.Uint64 // 熔断拒绝的回源次数
}

// Stats 为运行时统计快照，可据此观测缓存命中率与行情源健康度。
type Stats struct {
	// Hits 为缓存命中次数：查询结果直接来自缓存，未触碰行情源。
	Hits uint64
	// EmptyHits 为空标记命中次数：空标记存活期内不再回源（含失败标记
	// 与确认无数据的空结果标记）。
	EmptyHits uint64
	// Fetches 为实际执行回源次数（成功与失败都计入）。
	Fetches uint64
	// SourceErrors 为回源失败次数。
	SourceErrors uint64
	// CircuitRejects 为熔断打开期间被拒绝的回源次数。
	CircuitRejects uint64
}

// Stats 返回各项计数的当前快照；计数自实例创建起累计，不重置。
func (c *CacheClient) Stats() Stats {
	return Stats{
		Hits:           c.stats.hits.Load(),
		EmptyHits:      c.stats.emptyHits.Load(),
		Fetches:        c.stats.fetches.Load(),
		SourceErrors:   c.stats.sourceErrors.Load(),
		CircuitRejects: c.stats.circuitRejects.Load(),
	}
}

// NewCacheClient 接收外部创建的 `redis.Client` 与行情源 `gotdx.Client`，
// 并按给定选项构建缓存层。rds 与 cli 均不允许为 nil；选项为 nil 或未覆盖
// 的字段使用默认值（非法值回退默认）。实例无需显式关闭；传入的两个客户端
// 由外部创建、外部负责关闭，本包不触碰它们。
func NewCacheClient(rds *redis.Client, cli *gotdx.Client, opts ...Option) *CacheClient {
	if rds == nil {
		panic("cache: redis client is nil")
	}
	if cli == nil {
		panic("cache: gotdx client is nil")
	}
	o := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	// 非法值回退默认，避免阈值 0 导致熔断永久打开、TTL 0 立即过期、
	// 空前缀产生畸形键等异常行为。
	if o.KeyPrefix == "" {
		o.KeyPrefix = DefaultKeyPrefix
	}
	if o.EmptyTTL <= 0 {
		o.EmptyTTL = DefaultEmptyTTL
	}
	if o.LockTimeout <= 0 {
		o.LockTimeout = DefaultLockTimeout
	}
	if o.LiveTTL <= 0 {
		o.LiveTTL = DefaultLiveTTL
	}
	if o.CircuitFailThreshold <= 0 {
		o.CircuitFailThreshold = DefaultCircuitFailThreshold
	}
	if o.CircuitOpenTimeout <= 0 {
		o.CircuitOpenTimeout = DefaultCircuitOpenTimeout
	}
	return &CacheClient{
		opt:     o,
		rds:     rds,
		cli:     cli,
		breaker: newCircuitBreaker(o.CircuitFailThreshold, o.CircuitOpenTimeout),
	}
}

// lock 尝试获取分布式锁并执行回调：获取失败返回 errLockNotAcquired；
// 获取成功后执行回调，无论结果如何都在返回前释放锁。锁值使用随机
// token，释放时校验 token，防止误删其他进程新获取的锁。key 为完整
// 锁键，由调用方按类型段构造。
func (c *CacheClient) lock(ctx context.Context, key string, fn func() error) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	ok, err := c.rds.SetNX(ctx, key, token, c.opt.LockTimeout).Result()
	if err != nil {
		return err
	}
	if !ok {
		return errLockNotAcquired
	}
	defer func() {
		_ = c.unlock(ctx, key, token)
	}()
	return fn()
}

// unlock 释放分布式锁：仅当锁值仍为本进程持有的 token 时删除。
func (c *CacheClient) unlock(ctx context.Context, key, token string) error {
	return unlockScript.Run(ctx, c.rds, []string{key}, token).Err()
}

// randomToken 生成 16 字节随机 token 的十六进制文本，用于分布式锁
// 持有者标识。
func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// replyEnvelope 为普通键值缓存的实际存储结构：Version 用于格式迁移，
// Data 为单条应答 MessagePack 编码后的字节。
type replyEnvelope struct {
	Version int
	Data    []byte
}

// msgpackEncode 把任意可序列化值 MessagePack 编码；常规类型无需额外
// 注册。与 `gob` 相比编码更快、体积更小；注意 `map[string]any` 等
// 无目标类型字段内的数值解码时按 MessagePack 规范归一为 int64/uint64/
// float64，原始 float32/uint32 等窄类型不保留。
func msgpackEncode(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

// msgpackDecode 把 MessagePack 字节解码为指定类型。
func msgpackDecode[T any](b []byte) (T, error) {
	var v T
	if err := msgpack.Unmarshal(b, &v); err != nil {
		return v, err
	}
	return v, nil
}

// encodeReplyEnvelope 把应答信封 MessagePack 编码为可写入 `Redis` 的字节。
func encodeReplyEnvelope(env *replyEnvelope) ([]byte, error) {
	var buf bytes.Buffer
	if err := msgpack.NewEncoder(&buf).Encode(env); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decodeReplyEnvelope 从 `Redis` 字节还原应答信封。
func decodeReplyEnvelope(b []byte) (*replyEnvelope, error) {
	var env replyEnvelope
	if err := msgpack.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

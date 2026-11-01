// Package gotdx 通用行情订阅工厂：对任意数据源建立"轮询 diff 推送"订阅。
//
// 工厂与具体的查询方法解耦：调用方提供拉取函数（`Fetch`）与可选的变化判定
// （`Changed`），工厂负责按间隔轮询、快照对比与变化回调，因此 K 线、分时、逐笔、
// 指数、扩展市场等所有查询能力都可以用同一套订阅语义接入，无需为每种数据单独
// 实现推送循环。
//
// 基座能力：
//   - 高吞吐：相同 `Key` 的订阅合并到同一个轮询任务，一次拉取服务多个处理函数，
//     避免对服务器重复轮询；不同数据源的任务并行拉取。
//   - 高性能：轮询间隔每轮实时读取，间隔缩小无需重建任务循环；任务状态
//     （快照、处理函数列表）按锁分区管理，回调期间订阅增删不影响本次推送。
//   - 安全：拉取与回调均做异常隔离，单个订阅 panic 不影响其他订阅与进程；取消、
//     客户端关闭、工厂关闭均可及时收敛轮询循环，不泄漏 goroutine。
package gotdx

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// defaultSubscribeInterval 为未指定间隔时的默认轮询间隔：行情快照秒级刷新即可
// 满足大多数订阅场景，过短间隔会增加对服务器的轮询压力。
const defaultSubscribeInterval = 3 * time.Second

// Subscriber 是通用行情订阅工厂：绑定一个客户端连接，通过 `Subscribe` 对任意
// 数据源建立订阅；客户端 `Disconnect` 或调用 `Close` 时工厂内全部订阅自动停止。
type Subscriber struct {
	client *Client // 绑定的客户端，其 `done` 通道作为全局停止信号
	mu     sync.Mutex
	tasks  map[string]*subTask // 轮询任务表：键为数据源标识（`Key` 或独立订阅序号）
	seq    int                 // 独立订阅（未指定 `Key`）的唯一序号
}

// SubscribeSource 描述一个可订阅的数据源。
//
// `Key` 为可选的数据源标识：相同 `Key` 的订阅合并到同一个轮询任务，共享一次拉取
// （同一 `Key` 下快照类型必须一致，轮询间隔取所有订阅的最小值）；未指定 `Key`
// 时每次订阅独立轮询。`Fetch` 拉取最新快照，返回错误时跳过本轮轮询；`Changed`
// 判断两次快照是否不同（省略时默认深比较，即任何字段变化都视为变化）。
type SubscribeSource[T any] struct {
	Key     string                // 可选数据源标识，相同标识合并轮询
	Fetch   func() (T, error)     // 拉取最新快照，返回错误时跳过本轮轮询
	Changed func(old, new T) bool // 两次快照是否不同；`nil` 时使用深比较
}

// subHandler 为一个订阅处理函数的内部状态：取消与待推基线标记用原子变量，
// 回调线程与订阅线程可并发读写。
type subHandler struct {
	handler   func(any) bool // 包装后的处理函数，入参已断言为快照类型
	cancelled atomic.Bool    // 取消标记：处理函数返回 `false` 或调用取消函数时置位
	pending   atomic.Bool    // 待推基线标记：合并追加的订阅下轮无条件推送当前快照
}

// subTask 为一个数据源的轮询任务：多个订阅（`subHandler`）共享一次拉取。
//
// 锁分区：`f.mu` 保护任务表的增删与 `handlers` 切片的新增；`mu` 保护轮询状态
// （快照 `last`/`hasLast`）与拉取推送的串行执行（重建循环时新旧循环通过该锁
// 互斥，避免并发写快照）。两条锁路径互不嵌套，无死锁。
type subTask struct {
	mu       sync.Mutex   // 轮询状态锁：保护 `last`/`hasLast` 与单轮拉取推送的串行
	key      string       // 数据源标识
	typ      reflect.Type // 快照类型，用于校验同 `Key` 订阅类型一致
	fetch    func() (any, error)
	changed  func(old, new any) bool
	interval atomic.Int64  // 轮询间隔（纳秒），每轮实时读取，缩小即时生效
	handlers []*subHandler // 处理函数列表，增删由 `f.mu` 保护
	last     any           // 上次推送的快照，用于 diff
	hasLast  bool
	stopCh   chan struct{}
	stopOnce sync.Once // 停止通道只关闭一次
	gen      int       // 代际：重建循环时自增，退出清理仅回收当前代
}

// Subscriber 返回绑定当前客户端的通用订阅工厂：通过工厂可对报价、K 线、分时、
// 逐笔、指数、扩展市场等任意数据源建立订阅（见 `Subscriber.Subscribe`），客户端
// `Disconnect` 或调用 `Close` 时工厂内全部订阅自动停止。
func (client *Client) Subscriber() *Subscriber {
	return &Subscriber{client: client}
}

// Subscribe 对一个数据源建立订阅：注册后立即拉取一次并推送建立基线，此后按
// `interval` 轮询拉取，快照发生变化时回调 `handler`。
//
// 参数 `source` 描述数据源（`Fetch` 必填，`Key` 可选用于合并订阅）；`interval`
// 为轮询间隔，非正数时使用默认 3 秒；`handler` 接收变化后的最新快照，返回
// `false` 即取消本次订阅。返回取消函数，调用后停止该订阅的轮询与推送。拉取失败
// （网络波动或拉取函数 panic）自动跳过本轮并下轮重试，不中断订阅。
func (f *Subscriber) Subscribe[T any](source SubscribeSource[T], interval time.Duration, handler func(T) bool) (cancel func(), err error) {
	if source.Fetch == nil {
		return nil, errors.New("nil subscribe fetch")
	}
	if handler == nil {
		return nil, errors.New("nil subscribe handler")
	}
	if interval <= 0 {
		interval = defaultSubscribeInterval
	}
	if source.Changed == nil {
		source.Changed = func(old, new T) bool { return !reflect.DeepEqual(old, new) }
	}

	sh := &subHandler{
		handler: func(v any) bool { return handler(v.(T)) },
	}
	typ := reflect.TypeOf((*T)(nil)).Elem()

	f.mu.Lock()
	defer f.mu.Unlock()

	key := source.Key
	if key == "" {
		f.seq++
		key = fmt.Sprintf("__sub_%d", f.seq)
	}

	t := f.tasks[key]
	if t == nil {
		// 新建任务：独立循环负责基线推送与轮询。
		t = &subTask{
			key:      key,
			typ:      typ,
			fetch:    func() (any, error) { v, e := source.Fetch(); return any(v), e },
			changed:  func(old, new any) bool { return source.Changed(old.(T), new.(T)) },
			handlers: []*subHandler{sh},
			stopCh:   make(chan struct{}),
		}
		t.interval.Store(int64(interval))
		if f.tasks == nil {
			f.tasks = make(map[string]*subTask)
		}
		f.tasks[key] = t
		go f.runTask(t)
		return func() { f.cancelTask(t, sh) }, nil
	}

	// 同 `Key` 已存在任务：校验类型一致后合并订阅；新处理函数标记待推基线，
	// 即使快照当前无变化也会在下一轮无条件收到最新快照。
	if t.typ != typ {
		return nil, fmt.Errorf("subscribe key %q already bound to type %v", key, t.typ)
	}
	sh.pending.Store(true)
	t.handlers = append(t.handlers, sh)

	// 间隔更小或任务已停止时重建循环：旧循环经停止通道退出（幂等关闭），新循环
	// 复用既有快照状态，不重复推送基线。
	needRestart := isSubTaskStopped(t)
	if int64(interval) < t.interval.Load() {
		t.interval.Store(int64(interval))
		needRestart = true
	}
	if needRestart {
		t.stopOnce.Do(func() { close(t.stopCh) })
		t.stopCh = make(chan struct{})
		t.stopOnce = sync.Once{}
		t.gen++
		go f.runTask(t)
	}
	return func() { f.cancelTask(t, sh) }, nil
}

// Close 停止工厂内全部订阅并释放轮询循环；之后仍可继续调用 `Subscribe` 建立新订阅。
func (f *Subscriber) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, t := range f.tasks {
		t.stopOnce.Do(func() { close(t.stopCh) })
	}
	f.tasks = make(map[string]*subTask)
}

// cancelTask 标记订阅取消：置位处理函数取消标记；任务无活跃处理函数时关闭停止
// 通道，轮询循环随之退出并回收。
func (f *Subscriber) cancelTask(t *subTask, sh *subHandler) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if sh.cancelled.Swap(true) {
		return
	}
	if !f.hasAliveHandler(t) {
		t.stopOnce.Do(func() { close(t.stopCh) })
	}
}

// hasAliveHandler 判断任务是否仍有未取消的处理函数（需持有 `f.mu`）。
func (f *Subscriber) hasAliveHandler(t *subTask) bool {
	for _, h := range t.handlers {
		if !h.cancelled.Load() {
			return true
		}
	}
	return false
}

// isSubTaskStopped 判断任务停止通道是否已关闭（需持有 `f.mu` 或与停止通道的
// 关闭操作互斥）。
func isSubTaskStopped(t *subTask) bool {
	select {
	case <-t.stopCh:
		return true
	default:
		return false
	}
}

// runTask 单个数据源任务的轮询循环：任务首次运行（尚无快照）时先无条件推送基线，
// 此后按间隔轮询 diff 推送；停止通道关闭、客户端关闭或全部处理函数取消时退出。
// 退出时仅回收当前代际的任务表条目，避免重建代际误删。
func (f *Subscriber) runTask(t *subTask) {
	gen := t.gen
	defer func() {
		f.mu.Lock()
		if f.tasks[t.key] == t && t.gen == gen {
			delete(f.tasks, t.key)
		}
		f.mu.Unlock()
	}()

	if !f.pumpTask(t, true) {
		return
	}
	// 停止通道在 `Disconnect` 持 `client.mu` 关闭并置空，锁内取一次快照供本循环
	// 使用，避免与 `done` 字段的置空形成数据竞争。
	f.client.mu.Lock()
	done := f.client.done
	f.client.mu.Unlock()
	timer := time.NewTimer(time.Duration(t.interval.Load()))
	defer timer.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-done:
			return
		case <-timer.C:
			if !f.pumpTask(t, false) {
				return
			}
			// 每轮重读间隔：间隔缩小即时生效，无需重建循环。
			timer.Reset(time.Duration(t.interval.Load()))
		}
	}
}

// pumpTask 拉取一次快照并按 diff 推送：`force` 为真且尚无快照时无条件推送建立
// 基线（已有快照则退化为正常 diff）；快照无变化时不推送，待推基线的追加订阅除外。
// 返回 `false` 表示任务应退出（全部处理函数已取消）。
func (f *Subscriber) pumpTask(t *subTask, force bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if force {
		force = !t.hasLast // 已有快照时不再强制推送
	}
	v, err := t.safeFetch()
	if err != nil {
		return true // 拉取失败或拉取函数 panic，跳过本轮
	}
	changed := force || !t.hasLast || t.safeChanged(t.last, v)
	if changed {
		t.last = v
		t.hasLast = true
	}

	// 锁内拷贝活跃处理函数并压缩剔除已取消项，出锁回调：回调期间订阅增删不影响
	// 本次推送，同时收敛 `handlers` 切片，避免高频订阅-取消导致其无限增长。
	f.mu.Lock()
	hs := make([]*subHandler, 0, len(t.handlers))
	for _, h := range t.handlers {
		if !h.cancelled.Load() {
			hs = append(hs, h)
		}
	}
	t.handlers = hs
	f.mu.Unlock()

	for _, h := range hs {
		// 追加订阅（待推基线）：无条件推送当前快照建立基线，不参与 diff 判定。
		if h.pending.Swap(false) {
			if !t.safeHandler(h, v) {
				h.cancelled.Store(true)
			}
			continue
		}
		if !changed {
			continue
		}
		if !t.safeHandler(h, v) {
			h.cancelled.Store(true) // 处理函数返回 `false`，取消该订阅
		}
	}
	return len(hs) > 0
}

// safeFetch 包装拉取调用：拉取函数 panic 时按一次失败处理，不影响任务循环。
func (t *subTask) safeFetch() (v any, err error) {
	defer func() {
		if recover() != nil {
			v, err = nil, errors.New("subscribe fetch panic")
		}
	}()
	return t.fetch()
}

// safeChanged 包装变化判定：判定函数 panic 时视为无变化，跳过本轮推送。
func (t *subTask) safeChanged(old, new any) bool {
	defer func() { _ = recover() }()
	return t.changed(old, new)
}

// safeHandler 包装处理函数调用：处理函数 panic 时本轮跳过且不取消订阅（返回
// `true` 语义，保持订阅继续）。
func (t *subTask) safeHandler(h *subHandler, v any) (keep bool) {
	defer func() {
		if recover() != nil {
			keep = true
		}
	}()
	return h.handler(v)
}

package gotdx

import (
	"errors"
	"reflect"
	"sync"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

// ErrMarketCodeCount 表示批量请求时传入的市场（或分类）列表与证券代码列表长度不一致，由
// `makeStocks`、`makeExStocks` 在配对前校验长度时返回。
var ErrMarketCodeCount = errors.New("market code count error")

// execute 完成一次协议请求-响应流程：从池内借出一条连接（借出即移出池）、持该连接
// 串行锁执行收发，解析成功后归还池内；收发或解析失败时由池销毁该条并后台补建。
//
// 解析成功后记录本次借出连接地址为"最近成功响应的服务器"（`lastDataAddr`），后续借出
// 定向优先同地址连接，让请求稳定落回已验证节点；该地址仅在重平衡切换或连接丢弃后
// 自然失效。
//
// 空数据换节点：应答为空数据（通用判空规则见 `isEmptyReply`）时解除定向、累计该节点
// 空数据健康度，并并行探测池内其余连接取首个非空应答（重试条数沿用原最多 2 次上限）；
// 命中非空后重新定向并清零该节点空数据计数；并行探测仍空（该日期确实无成交）返回首次
// 应答。部分服务器对较早历史日期数据不全，多地址池下换节点可命中数据完整的服务器
// （老库固定单节点必然命中其节点，新库以换节点重试达成同等稳定）。`Options.DisableEmptyNodeSwitch`
// 开启后跳过上述换节点流程，空应答直接返回首次结果。
//
// 参数 `protocol` 为待执行的协议对象，泛型 `T` 为应答类型；池内暂无连接（未连接或全部被
// 借出/丢弃待补建）或收发、解析失败时返回 `T` 零值与错误。
func (client *Client) execute[T any](protocol proto.Protocol[T]) (T, error) {
	var zero T
	reply, addr, err := client.executeOnce(protocol)
	if err != nil {
		return zero, err
	}
	if isEmptyReply(reply) {
		// 单连接模式或显式关闭换节点：空应答直接返回首次结果，不解除定向、不并行探测
		// 其他连接、不累计健康度，行为对齐老库固定单节点；断连自愈仍由心跳与补建负责。
		if client.opt.DisableEmptyNodeSwitch {
			return reply, nil
		}
		// 空数据：解除对当前节点的定向并计入健康度，随后并行探测其余连接。
		client.clearDataAddr(addr)
		client.recordNodeHealthEmpty(addr)
		if nonEmpty, nonEmptyAddr, found := client.parallelEmptyRetry(protocol); found {
			client.recordNodeHealthOK(nonEmptyAddr)
			client.setDataAddr(nonEmptyAddr)
			return nonEmpty, nil
		}
		// 并行探测仍空：返回首次应答（该日期可能确实无成交）；若池内已有节点被健康度
		// 降权，后台触发一次强制全量探活重平衡，尽快迁出数据不全节点，不必等到下一次
		// 定时 tick。重平衡自身由串行锁保护，与定时 tick 并发时自动跳过。
		if client.hasDegradedNode() {
			go client.rebalancePool(true)
		}
		return reply, nil
	}
	// 本次连接即为可用节点：清零其空数据计数并记录供定向优先复用。
	client.recordNodeHealthOK(addr)
	client.setDataAddr(addr)
	return reply, nil
}

// parallelEmptyRetry 在首查空数据后并行探测池内其余连接，取首个非空应答。
//
// 方案决策：原实现串行最多 2 次 `executeOnce`，慢节点上最坏尾延迟叠加 3 个 RTT；改为
// 并发借出池内空闲连接同时探测，首个非空应答立即胜出，尾延迟收敛到约 1 个 RTT。并发
// 探测条数沿用原最多 2 次的重试上限；各 `executeOnce` 自行借出/归还连接，连接串行锁
// 保证帧边界不被并发破坏。在途探测在 `WaitGroup` 中收尾，连接由 `executeOnce` 自行
// 归还或销毁，不泄漏。全部探测出错或仍空时 `found` 为 `false`，由调用方返回首次空应答。
func (client *Client) parallelEmptyRetry[T any](protocol proto.Protocol[T]) (T, string, bool) {
	var zero T
	const maxEmptyRetry = 2
	type retryResult struct {
		reply T
		addr  string
		err   error
		empty bool
	}
	results := make(chan retryResult, maxEmptyRetry)
	var wg sync.WaitGroup
	for i := 0; i < maxEmptyRetry; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, addr, err := client.executeOnce(protocol)
			results <- retryResult{reply: r, addr: addr, err: err, empty: err == nil && isEmptyReply(r)}
		}()
	}
	go func() { wg.Wait(); close(results) }()
	for res := range results {
		if res.err != nil {
			continue
		}
		if res.empty {
			client.recordNodeHealthEmpty(res.addr)
			continue
		}
		return res.reply, res.addr, true
	}
	return zero, "", false
}

// isEmptyReply 通过反射报告协议应答是否为空数据：历史、分时、逐笔类协议的应答
// 结构统一携带 `Count uint16` 字段，`Count` 为零即无任何记录；MAC 偏移 K 线应答
// 无 `Count` 字段，按 `Returned` 字段判空。反射仅读取一个 `uint16` 字段，单次
// 开销可忽略。应答类型不含上述字段时按非空处理（不触发换节点）。
func isEmptyReply(v any) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	if f := rv.FieldByName("Count"); f.IsValid() && f.Kind() == reflect.Uint16 {
		return f.Uint() == 0
	}
	if f := rv.FieldByName("Returned"); f.IsValid() && f.Kind() == reflect.Uint32 {
		return f.Uint() == 0
	}
	return false
}

// executeOnce 执行单次协议请求-响应：借出连接、收发、解析并归还/销毁连接，由
// `execute` 编排调用。返回值依次为应答、本次借出连接的实际地址与错误（无错误时
// 为 `nil`）。
func (client *Client) executeOnce[T any](protocol proto.Protocol[T]) (T, string, error) {
	var zero T
	pc, err := client.acquire()
	if err != nil {
		return zero, "", err
	}
	pc.mu.Lock()
	header, payload, err := client.exchange(pc.conn, protocol)
	if err == nil {
		err = protocol.ParseResponse(header, payload)
	}
	if err != nil {
		// 收发或解析失败后该连接帧边界不可信，销毁并交后台补建。
		client.discard(pc)
		pc.mu.Unlock()
		return zero, "", err
	}
	pc.mu.Unlock()
	client.release(pc)
	return protocol.Response(), pc.addr, nil
}

func makeCode6(code string) [6]byte {
	var out [6]byte
	copy(out[:], code)
	return out
}

// healthDowngradeThreshold 为节点连续空数据降权阈值：达到该次数后定向借出跳过该地址、
// 回退轮询。取值保守：偶发单次空数据（该日期确实无成交）不触发降权，只有同一节点连续
// 返回空数据才判定为数据不全并降低其定向优先级。
const healthDowngradeThreshold = 3

// nodeHealth 记录单个服务器地址的数据完整度健康度：`emptyStreak` 为连续空数据次数，
// 一次非空成功即清零；计数达到 `healthDowngradeThreshold` 后该地址不再参与借出定向。
type nodeHealth struct {
	emptyStreak int
}

// setDataAddr 记录指定地址为最近成功响应的服务器，供借出定向优先复用。
func (client *Client) setDataAddr(addr string) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if addr != "" {
		client.lastDataAddr = addr
	}
}

// clearDataAddr 清除对指定地址的定向记录：该地址返回空数据（或已失效）时调用，
// 使后续借出回退轮询，避免请求继续固定在数据不全的节点上。
func (client *Client) clearDataAddr(addr string) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.lastDataAddr == addr {
		client.lastDataAddr = ""
	}
}

// recordNodeHealthOK 记录指定地址本次返回非空数据：清零其连续空数据计数，节点恢复定向
// 资格；空地址忽略。映射惰性初始化，健康度统计只影响内部定向排序，不改变公开 API。
func (client *Client) recordNodeHealthOK(addr string) {
	if addr == "" {
		return
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.health == nil {
		client.health = make(map[string]*nodeHealth)
	}
	h := client.health[addr]
	if h == nil {
		h = &nodeHealth{}
		client.health[addr] = h
	}
	h.emptyStreak = 0
}

// recordNodeHealthEmpty 记录指定地址本次返回空数据：累加连续空数据计数；空地址忽略。
func (client *Client) recordNodeHealthEmpty(addr string) {
	if addr == "" {
		return
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.health == nil {
		client.health = make(map[string]*nodeHealth)
	}
	h := client.health[addr]
	if h == nil {
		h = &nodeHealth{}
		client.health[addr] = h
	}
	h.emptyStreak++
}

// nodeHealthDegradedLocked 报告指定地址是否已被健康度降权（连续空数据达到阈值）；
// 调用方须已持有池锁。无健康记录（从未返回空数据）视为未降权，保持原定向行为。
func (client *Client) nodeHealthDegradedLocked(addr string) bool {
	if addr == "" || client.health == nil {
		return false
	}
	h := client.health[addr]
	return h != nil && h.emptyStreak >= healthDowngradeThreshold
}

// hasDegradedNode 报告当前是否存在被健康度降权的节点，供全空重试后判断是否需要触发
// 后台强制重平衡；空映射或无降权节点返回 `false`，保持原行为不发起额外探活。
func (client *Client) hasDegradedNode() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	for _, h := range client.health {
		if h.emptyStreak >= healthDowngradeThreshold {
			return true
		}
	}
	return false
}

func makeCode9(code string) [9]byte {
	var out [9]byte
	copy(out[:], code)
	return out
}

func makeCode23(code string) [23]byte {
	var out [23]byte
	copy(out[:], code)
	return out
}

func makeCode22(code string) [22]byte {
	var out [22]byte
	copy(out[:], code)
	return out
}

func makeFixed40(value string) [40]byte {
	var out [40]byte
	copy(out[:], value)
	return out
}

func makeFixed80(value string) [80]byte {
	var out [80]byte
	copy(out[:], value)
	return out
}

func makeFixed300(value string) [300]byte {
	var out [300]byte
	copy(out[:], value)
	return out
}

func makeFixed43(value string) [43]byte {
	var out [43]byte
	copy(out[:], value)
	return out
}

// makeStocks 把市场与代码按下标两两配对组装成主站证券列表。
//
// 参数 `markets` 为市场代码序列，取值见 `types.Market*`；参数 `codes` 为六位证券代码序列，与市场
// 序列按位置一一对应。两者长度不一致时返回 `nil` 与 `ErrMarketCodeCount`。
func makeStocks(markets []uint8, codes []string) ([]proto.Stock, error) {
	if len(markets) != len(codes) {
		return nil, ErrMarketCodeCount
	}

	stocks := make([]proto.Stock, 0, len(markets))
	for i, market := range markets {
		stocks = append(stocks, proto.Stock{
			Market: market,
			Code:   codes[i],
		})
	}
	return stocks, nil
}

// makeExStocks 把扩展分类与代码按下标两两配对组装成扩展站证券列表。
//
// 参数 `categories` 为扩展市场类别序列，取值见 `types.ExMarket*` 与 `types.ExCategory*`；参数
// `codes` 为扩展站证券代码序列，与分类序列按位置一一对应。两者长度不一致时返回 `nil` 与
// `ErrMarketCodeCount`。
func makeExStocks(categories []uint8, codes []string) ([]proto.ExStock, error) {
	if len(categories) != len(codes) {
		return nil, ErrMarketCodeCount
	}

	stocks := make([]proto.ExStock, 0, len(categories))
	for i, category := range categories {
		stocks = append(stocks, proto.ExStock{
			Category: category,
			Code:     codes[i],
		})
	}
	return stocks, nil
}

// quotesSortReverse 把布尔排序方向转换为协议方向码。
//
// 参数 `sortType` 为排序字段编号，取值见 `types.Sort*`；参数 `reverse` 为 `true` 表示降序。代码排序
// （`types.SortCode`）不区分升降序，恒返回 0；其余字段 `reverse` 为 `false` 返回 1（升序），为
// `true` 返回 2（降序）。
func quotesSortReverse(sortType uint16, reverse bool) uint16 {
	if sortType == types.SortCode {
		return 0
	}
	if reverse {
		return 2
	}
	return 1
}

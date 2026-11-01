// Command stock_index_tools 演示指数实时信息、指数动量序列、指数 K 线以及分时采样价格四类指数相关接口的查询与打印。
//
// 运行示例：`go run ./examples/stock_index_tools`
package main

import (
	"fmt"
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 市场 `types.MarketSH`（沪市），指数代码 `000001`；返回指数实时概况，含点位、开高低收与上涨、下跌家数。
	info, err := client.StockIndexInfo(types.MarketSH.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("index_info code=%s time=%s close=%.2f open=%.2f high=%.2f low=%.2f up=%d down=%d",
		info.Code, info.ServerTime, info.Close, info.Open, info.High, info.Low, info.UpCount, info.DownCount)

	// 查询同一指数的动量数值序列；返回整段数值序列，示例仅打印序列长度与最后一个取值。
	momentum, err := client.StockIndexMomentum(types.MarketSH.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("index_momentum count=%d last=%d", len(momentum), momentum[len(momentum)-1])

	// `category` 取 `types.KLINE_TYPE_DAILY`（日 K）；`start=0` 为起始偏移条数，`count=1` 为请求条数；
	// 该接口固定周期倍数为 1、不复权。
	bars, err := client.GetIndexBars(types.KLINE_TYPE_DAILY, types.MarketSH.Uint8(), "000001", 0, 1)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range bars.List {
		log.Printf("index_bar time=%s close=%.2f high=%.2f low=%.2f", item.DateTime, item.Close, item.High, item.Low)
	}

	// 分时采样接口在部分主站主机上不可用，先逐个主机尝试再决定是否打印结果。
	sampling, err := getSampling()
	if err != nil {
		log.Printf("sampling unavailable: %v", err)
	} else {
		// 采样序列可能不足 10 个点，`min(10, len(...))` 防止切片越界，仅打印前 10 个价格点。
		for i, price := range sampling[:min(10, len(sampling))] {
			log.Printf("sampling index=%d price=%.2f", i, price)
		}
	}
}

// min 返回整数 a 与 b 中较小的一个。
//
// 参数 a、b 为待比较的两个整数；返回值为二者较小值，用于在切片截断时把长度限制在有效范围内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getSampling 逐个尝试主站主机查询分时采样价格序列，遇到首个可用主机即返回其结果。
//
// 本函数无入参；返回值 `[]float64` 为按时间顺序排列的采样价格序列，错误值为任一主机连接或协议失败时
// 得到的最后一次错误。遍历 `exampleutil.MainHosts()` 中的主站主机逐个短连接尝试，某个主机成功即返回；
// 所有主机均失败时返回最后一次主机的错误；未配置任何主站主机时返回“未配置主站主机”兜底错误。
func getSampling() ([]float64, error) {
	var lastErr error
	for _, host := range exampleutil.MainHosts() {
		client := exampleutil.NewMainClientForHost(host)
		reply, err := client.StockChartSampling(types.MarketSZ.Uint8(), "000001")
		_ = client.Disconnect()
		if err == nil {
			return reply, nil
		}
		lastErr = fmt.Errorf("%s sampling failed: %w", host, err)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no main quote hosts configured")
	}
	return nil, lastErr
}

// Command stock_lowlevel_quote 演示不经过统一封装，直接调用低层协议依次完成握手连接、证券计数、列表、快照、分时与逐笔查询。
//
// 运行示例：`go run ./examples/stock_lowlevel_quote`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 第一步：建立连接并读取握手返回的服务器信息。
	hello, err := client.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("connected info=%s", hello.Info)

	// 第二步：`market` 取 `types.MarketSZ`（深市），返回服务端统计的该市场证券总数。
	count, err := client.GetSecurityCount(types.MarketSZ.Uint8())
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("security_count=%d", count.Count)

	// 第三步：`start=0` 为起始偏移序号（从 0 开始），`count=200` 为本页请求条数，按区间拉取深市证券列表。
	list, err := client.GetSecurityListRange(types.MarketSZ.Uint8(), 0, 200)
	if err != nil {
		log.Fatalln(err)
	}
	if len(list.List) > 0 {
		log.Printf("security_list first=%s/%s count=%d", list.List[0].Code, list.List[0].Name, len(list.List))
	}

	// 第四步：市场序列与代码序列按下标一一对应，依次为深市 `000001`、沪市 `600000`，批量取回低层行情快照。
	quotes, err := client.GetSecurityQuotes([]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()}, []string{"000001", "600000"})
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range quotes.List {
		log.Printf("security_quote code=%s price=%.2f open=%.2f high=%.2f low=%.2f", item.Code, item.Price, item.Open, item.High, item.Low)
	}

	// 第五步：查询深市 `000001` 的当日分时数据；`min(10, len(...))` 防止切片越界，仅打印前 10 个时点。
	minute, err := client.GetMinuteTimeData(types.MarketSZ.Uint8(), "000001")
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range minute.List[:min(10, len(minute.List))] {
		log.Printf("minute price=%.2f avg=%.4f vol=%d", item.Price, item.Avg, item.Vol)
	}

	// 第六步：查询深市 `000001` 当日逐笔成交，`start=0` 为起始成交序号，`count=10` 为本页请求笔数。
	trans, err := client.GetTransactionData(types.MarketSZ.Uint8(), "000001", 0, 10)
	if err != nil {
		log.Fatalln(err)
	}
	for _, item := range trans.List {
		log.Printf("transaction time=%s price=%.2f vol=%d num=%d action=%d", item.Time, item.Price, item.Vol, item.Num, item.BuyOrSell)
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

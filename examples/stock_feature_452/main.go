// Command stock_feature_452 演示调用试验协议 `0x452` 拉取证券特征记录，并打印市场、代码与两个待确认语义的特征数值。
//
// 运行示例：`go run ./examples/stock_feature_452`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 试验协议 `0x452` 不区分市场入参；`start=0` 为起始偏移序号（从 0 开始），`count=10` 为本页请求条数。
	// 返回特征条目；其中 `P1`、`P2` 两个数值字段语义待确认，示例仅按原样打印，不推断其口径。
	items, err := client.StockFeature452(0, 10)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("feature_452 market=%d code=%s p1=%.4f p2=%.4f",
			item.Market, item.Code, item.P1, item.P2)
	}
}

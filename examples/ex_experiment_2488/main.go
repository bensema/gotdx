// Command ex_experiment_2488 演示试验协议 `0x2488` 返回的数据记录，以美股 `TSLA` 为例逐条打印；
// 请求模式与各字段的确切语义待确认。
//
// 运行示例：`go run ./examples/ex_experiment_2488`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `category` 为美股类别，`code` 为 `TSLA`；`mode` 为协议内部模式取值 55，其取值含义与
	// 响应各字段语义待确认。
	items, err := client.ExExperiment2488(types.ExCategoryUSStock, "TSLA", 55)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("ex_2488 id=%d values=%v", item.ID, item.Values)
	}
}

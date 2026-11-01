// Command ex_list 演示通过 ExList 分页获取扩展市场证券列表，从偏移 0 开始打印前 20 条。
//
// 运行示例：`go run ./examples/ex_list`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `start` 为起始偏移序号 0，`count` 为本页请求 20 条证券列表记录。
	items, err := client.ExList(0, 20)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("market=%d category=%d code=%s name=%s", item.Market, item.Category, item.Code, item.Name)
	}
}

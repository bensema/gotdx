// Command ex_list_extra 演示通过 ExListExtra 获取扩展市场附加列表记录并逐条打印；前两个参数的确切语义待确认。
//
// 运行示例：`go run ./examples/ex_list_extra`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	// 参数 `a`、`b` 为协议附加定位参数，确切语义待确认，此处先传 0 演示调用；`count` 为本页请求 10 条记录。
	items, err := client.ExListExtra(0, 0, 10)
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("ex_list_extra category=%d code=%s flag=%d values=%v",
			item.Category, item.Code, item.Flag, item.Values)
	}
}

// Command ex_table_detail 演示通过 ExTableDetail 拉取扩展市场表格明细文本，按逗号拆分后打印前 20 个非空条目。
//
// 运行示例：`go run ./examples/ex_table_detail`
package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	content, err := client.ExTableDetail()
	if err != nil {
		log.Fatalln(err)
	}

	// 明细文本以逗号分隔字段；用 `min(20, len(items))` 兜底截取前 20 条，空条目直接跳过。
	items := strings.Split(content, ",")
	for _, item := range items[:min(20, len(items))] {
		if item == "" {
			continue
		}
		log.Println(item)
	}
}

// min 返回 `a`、`b` 中较小的整数，用于把截取条数限制在切片实际长度以内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

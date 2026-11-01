// Command ex_table 演示通过 ExTable 拉取扩展市场表格文本，按换行拆分后打印前 10 个非空行。
//
// 运行示例：`go run ./examples/ex_table`
package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	content, err := client.ExTable()
	if err != nil {
		log.Fatalln(err)
	}

	// 表格文本按行组织；用 `min(10, len(lines))` 兜底截取前 10 行，去除首尾空白后跳过空行。
	lines := strings.Split(content, "\n")
	for _, line := range lines[:min(10, len(lines))] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		log.Println(line)
	}
}

// min 返回 `a`、`b` 中较小的整数，用于把截取条数限制在切片实际长度以内，避免越界。
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

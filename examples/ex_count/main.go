// Command ex_count 演示通过 ExCount 获取扩展市场内的标的总数并打印。
//
// 运行示例：`go run ./examples/ex_count`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewExClient()
	defer client.Disconnect()

	count, err := client.ExCount()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("ex_count=%d", count)
}

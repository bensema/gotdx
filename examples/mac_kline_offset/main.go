// Command mac_kline_offset 演示通过 MACKLineOffset 获取 MAC K 线偏移信息，从偏移 0 开始按默认条数请求，
// 打印服务端返回的总数与本次返回条数。
//
// 运行示例：`go run ./examples/mac_kline_offset`
package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 参数 `offset` 为起始 K 线条数偏移 0；`count` 显式传入内置默认条数 `DefaultMACKLineOffsetCount`；
	// 返回 `Total` 为 K 线总数，`Returned` 为本次实际返回条数。
	reply, err := client.MACKLineOffset(0, gotdx.DefaultMACKLineOffsetCount)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("mac_kline_offset total=%d returned=%d", reply.Total, reply.Returned)
}

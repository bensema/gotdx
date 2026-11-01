// Command mac_server_info 演示查询 MAC 行情服务器的交易日、会话与市场参数等基本信息并打印。
//
// 运行示例：`go run ./examples/mac_server_info`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 该请求不携带业务参数；返回当前交易日、上一交易日、会话计数、标志位与市场参数等服务端状态。
	reply, err := client.MACServerInfo()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("mac_server_info today=%s last_trading_day=%s sessions1=%v sessions2=%v flag=%d params=%d/%d",
		reply.Today, reply.LastTradingDay, reply.Sessions1, reply.Sessions2, reply.Flag, reply.MarketParam1, reply.MarketParam2)
}

// Command mac_symbol_quotes 演示通过 MAC 协议批量查询沪深两只证券的动态报价并打印命中字段与取值。
//
// 运行示例：`go run ./examples/mac_symbol_quotes`
package main

import (
	"log"

	"github.com/bensema/gotdx"
	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 市场序列与代码序列按下标一一对应：深市 `000001`、沪市 `600000`；第三参为 20 字节字段位图，
	// 声明每只标的需要返回哪些动态报价字段，传全零时由服务端使用内置默认位图。
	reply, err := client.MACSymbolQuotes(
		[]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()},
		[]string{"000001", "600000"},
		gotdx.DefaultMACSymbolQuotesFieldBitmap(),
	)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("mac_symbol_quotes total=%d count=%d fields=%d bitmap=%x",
		reply.Total, reply.Count, len(reply.ActiveFields), reply.FieldBitmap)

	// 返回证券可能不足 5 只，`min(5, len(...))` 防止切片长度越界，仅打印前 5 只的代码、名称与常用字段。
	for _, item := range reply.Stocks[:min(5, len(reply.Stocks))] {
		log.Printf("market=%d symbol=%s name=%s close=%v pre_close=%v turnover=%v pe_ttm=%v",
			item.Market,
			item.Symbol,
			item.Name,
			item.Values["close"],
			item.Values["pre_close"],
			item.Values["turnover"],
			item.Values["pe_ttm"],
		)
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

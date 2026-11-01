// Command mac_board_list 演示先查询全部类型板块的总数，再按该总数拉取 MAC 板块列表并逐行打印。
//
// 运行示例：`go run ./examples/mac_board_list`
package main

import (
	"fmt"
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/types"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()
	// 参数 `boardType` 取全部板块类型 `BoardTypeAll`；`MACBoardCount` 返回该类型板块的服务端总数。
	BoardType := types.BoardTypeAll
	count, _err := client.MACBoardCount(BoardType)
	if _err != nil {
		log.Fatalln(_err)
	}

	// 把板块总数作为 `count` 上限传入；接口内部按固定单页大小自动分页，末页自动收缩。
	items, err := client.MACBoardList(BoardType, uint32(count))
	if err != nil {
		log.Fatalln(err)
	}

	for _, item := range items {
		log.Printf("mac_board code=%s name=%s price=%.2f rise_speed=%.2f symbol=%s/%s",
			item.Code, item.Name, item.Price, item.RiseSpeed, item.SymbolCode, item.SymbolName)
	}
	fmt.Println(count)

}

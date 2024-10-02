# gotdx
通达信股票行情API golang版


## API
- Connect 连接券商行情服务器
- Disconnect 断开服务器
- GetSecurityCount 获取指定市场内的证券数目
- GetSecurityQuotes 获取盘口五档报价
- GetSecurityList 获取市场内指定范围内的所有证券代码
- GetSecurityBars 获取股票K线
- GetIndexBars 获取指数K线
- GetMinuteTimeData 获取分时图数据
- GetHistoryMinuteTimeData 获取历史分时图数据
- GetTransactionData 获取分时成交
- GetHistoryTransactionData 获取历史分时成交


## Example
```go
package main

import (
	"github.com/bensema/gotdx"
	"log"
)

func main() {
	tdx := gotdx.New(gotdx.WithTCPAddress("119.147.212.81:7709"))
	_, err := tdx.Connect()
	if err != nil {
		log.Fatalln(err)
	}
	defer tdx.Disconnect()

	reply, err := tdx.GetSecurityQuotes([]uint8{gotdx.MarketSh, gotdx.MarketSz}, []string{"000001", "600008"})
	if err != nil {
		log.Println(err)
	}

	for _, obj := range reply.List {
		log.Printf("%+v", obj)
	}
}


```


## Test
```bash
 go test
```
// Command stock_server_info 演示主站握手、服务器心跳、服务器信息、主站公告与交易所公告等服务端信息接口的查询与预览打印。
//
// 运行示例：`go run ./examples/stock_server_info`
package main

import (
	"log"
	"strings"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		log.Fatalln(err)
	}

	// 心跳请求不携带业务参数；`Date` 为服务端应答中记录的打包日期整数。
	heartbeat, err := client.GetServerHeartbeat()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("heartbeat date=%d", heartbeat.Date)

	// 服务信息请求不携带业务参数；`TimeNow` 为服务端当前时间，`Region` 为区域编号，`MaybeSwitch` 为语义
	// 待确认的开关位字段，`Info` 为服务端文案；三个保留字段 `Unknown1/2/3` 语义待确认，仅原样打印。
	info, err := client.GetServerInfo()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("server_info now=%s region=%d switch=%d info=%q",
		info.TimeNow, info.Region, info.MaybeSwitch, info.Info)
	log.Printf("server_info unknown1=%v unknown2=%v unknown3=%v",
		info.Unknown1, info.Unknown2, info.Unknown3)

	// 公告请求不携带业务参数；`HasContent` 标识是否包含正文，`preview(..., 80)` 把正文截断到 80 字节后预览。
	announcement, err := client.GetAnnouncement()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("announcement has_content=%t title=%q preview=%q",
		announcement.HasContent, announcement.Title, preview(announcement.Content, 80))

	// 交易所公告请求不携带业务参数；`Version` 为公告版本号，正文同样截断到 80 字节后预览。
	exchangeAnnouncement, err := client.GetExchangeAnnouncement()
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("exchange_announcement version=%d preview=%q",
		exchangeAnnouncement.Version, preview(exchangeAnnouncement.Content, 80))
}

// preview 去掉文本首尾空白后截断到指定长度，用于在日志中打印公告正文预览。
//
// 参数 `text` 为待预览的原始文本；参数 `limit` 为预览保留的最大字节数。文本未超长时原样返回；超长时按
// `limit` 字节截断（未做字符边界处理，多字节字符可能被切在中间），返回截断后的预览串。
func preview(text string, limit int) string {
	text = strings.TrimSpace(text)
	if len(text) <= limit {
		return text
	}
	return text[:limit]
}

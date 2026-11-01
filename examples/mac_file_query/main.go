// Command mac_file_query 演示 MAC 文件两步操作：先查询 `StockInfo.dat` 的元信息（偏移、大小、标志、哈希），
// 再按元信息中的大小循环下载完整文件并打印字节数。
//
// 运行示例：`go run ./examples/mac_file_query`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
)

func main() {
	client := exampleutil.NewMACClient()
	defer client.Disconnect()

	// 第一步：查询文件元信息。`filename` 为协议文件名 `StockInfo.dat`，内部按定长 70 字节字段填充；
	// `offset` 为协议起始偏移，此处传 0；返回的 `Size` 作为后续下载的总大小上限。
	meta, err := client.MACFileList("StockInfo.dat", 0)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("mac_file_list offset=%d size=%d flag=%d hash=%s", meta.Offset, meta.Size, meta.Flag, meta.Hash)

	// 第二步：`index` 为文件分片序号，协议从 1 开始；`size` 直接取上一步元信息中的大小；接口按固定
	// 块大小循环下载，服务器返回空块或达到大小上限时结束。
	content, err := client.MACDownloadFullFile("StockInfo.dat", 1, meta.Size)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("mac_file_download bytes=%d", len(content))
}

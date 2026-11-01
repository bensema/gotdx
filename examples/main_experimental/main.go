// Command main_experimental 演示主站若干语义未公开的试验协议：逐条发起请求，打印应答长度与原始十六进制预览。
//
// 运行示例：`go run ./examples/main_experimental`
package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/proto"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	// 闭包 `run` 统一执行一次试验协议调用。参数 `name` 为日志中展示的协议名称；参数 `call` 为无业务入参、
	// 返回原始应答与错误的协议闭包。调用失败时仅打印错误；成功时打印应答长度，并用 `PreviewHex` 把原始
	// 十六进制数据截断到 120 个字符后预览。
	run := func(name string, call func() (*proto.RawDataReply, error)) {
		reply, err := call()
		if err != nil {
			log.Printf("%s err=%v", name, err)
			return
		}
		log.Printf("%s len=%d hex=%s", name, reply.Length, exampleutil.PreviewHex(reply.Hex, 120))
	}

	// 依次调用下列主站试验协议；各协议业务语义未公开，示例不做字段级解析。
	run("todo_b", func() (*proto.RawDataReply, error) { return client.MainTodoB() })
	run("todo_fde", func() (*proto.RawDataReply, error) { return client.MainTodoFDE() })
	run("client_264b", func() (*proto.RawDataReply, error) { return client.MainClient264B() })
	run("client_26ac", func() (*proto.RawDataReply, error) { return client.MainClient26AC() })
	run("client_26ad", func() (*proto.RawDataReply, error) { return client.MainClient26AD() })
	run("client_26ae", func() (*proto.RawDataReply, error) { return client.MainClient26AE() })
	run("client_26b1", func() (*proto.RawDataReply, error) { return client.MainClient26B1() })
}

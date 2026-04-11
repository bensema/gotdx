package main

import (
	"log"

	"github.com/bensema/gotdx/examples/internal/exampleutil"
	"github.com/bensema/gotdx/proto"
)

func main() {
	client := exampleutil.NewMainClient()
	defer client.Disconnect()

	run := func(name string, call func() (*proto.RawDataReply, error)) {
		reply, err := call()
		if err != nil {
			log.Printf("%s err=%v", name, err)
			return
		}
		log.Printf("%s len=%d hex=%s", name, reply.Length, exampleutil.PreviewHex(reply.Hex, 120))
	}

	run("todo_b", func() (*proto.RawDataReply, error) { return client.MainTodoB() })
	run("todo_fde", func() (*proto.RawDataReply, error) { return client.MainTodoFDE() })
	run("client_264b", func() (*proto.RawDataReply, error) { return client.MainClient264B() })
	run("client_26ac", func() (*proto.RawDataReply, error) { return client.MainClient26AC() })
	run("client_26ad", func() (*proto.RawDataReply, error) { return client.MainClient26AD() })
	run("client_26ae", func() (*proto.RawDataReply, error) { return client.MainClient26AE() })
	run("client_26b1", func() (*proto.RawDataReply, error) { return client.MainClient26B1() })
}

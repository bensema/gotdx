package gotdx

import (
	"fmt"
	"os"
	"testing"

	"github.com/bensema/gotdx/types"
)

// newClient 连接内置固定测试地址并返回客户端。无入参；返回已尝试完成握手的 `Client`，
// 连接失败时仅打印错误而不中断测试，由后续请求断言暴露问题。
func newClient() *Client {
	tdx := New(WithTCPAddress("124.71.187.122:7709"))
	reply, err := tdx.Connect()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(reply.Info)
	return tdx
}

// requireIntegration 控制集成测试是否执行。参数 `t` 用于在跳过时终止当前用例；
// 仅当环境变量 `GOTDX_INTEGRATION=1` 时继续执行，否则调用 `t.Skip` 跳过。
func requireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("GOTDX_INTEGRATION") != "1" {
		t.Skip("set GOTDX_INTEGRATION=1 to run integration tests")
	}
}

func Test_tdx_Connect(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ Connect ================")
	tdx := New(WithTCPAddress("124.71.187.122:7709"))
	defer tdx.Disconnect()
	reply, err := tdx.Connect()
	if err != nil {
		t.Errorf("error:%s", err)
	}
	fmt.Println(reply.Info)
}

func Test_tdx_GetSecurityCount(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetSecurityCount ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityCount(types.MarketSH.Uint8())
	if err != nil {
		t.Errorf("error:%s", err)
	}
	fmt.Println(reply.Count)
}

func Test_tdx_GetSecurityQuotes(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetSecurityQuotes ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityQuotes([]uint8{types.MarketSZ.Uint8()}, []string{"002062"})
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetSecurityList(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetSecurityList ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityList(types.MarketSH.Uint8(), 0)
	if err != nil {
		t.Errorf("error:%s", err)
	}
	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetSecurityBars(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetSecurityBars ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityBars(types.KLINE_TYPE_RI_K, types.MarketSZ.Uint8(), "000001", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetIndexBars(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetIndexBars ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetIndexBars(types.KLINE_TYPE_RI_K, types.MarketSH.Uint8(), "000001", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetMinuteTimeData(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetMinuteTimeData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetMinuteTimeData(types.MarketSZ.Uint8(), "159607")
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetHistoryMinuteTimeData(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetHistoryMinuteTimeData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetHistoryMinuteTimeData(20220511, types.MarketSZ.Uint8(), "159607")
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetTransactionData(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetTransactionData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetTransactionData(types.MarketSZ.Uint8(), "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetHistoryTransactionData(t *testing.T) {
	requireIntegration(t)
	fmt.Println("================ GetHistoryTransactionData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetHistoryTransactionData(20230922, types.MarketSZ.Uint8(), "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

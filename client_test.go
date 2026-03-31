package gotdx

import (
	"fmt"
	"github.com/bensema/gotdx/proto"
	"os"
	"testing"
)

func newClient() *Client {
	tdx := New(WithTCPAddress("124.71.187.122:7709"))
	reply, err := tdx.Connect()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(reply.Info)
	return tdx
}

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
	reply, err := tdx.GetSecurityCount(MarketSH)
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
	reply, err := tdx.GetSecurityQuotes([]uint8{MarketSZ}, []string{"002062"})
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
	reply, err := tdx.GetSecurityList(MarketSH, 0)
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
	reply, err := tdx.GetSecurityBars(proto.KLINE_TYPE_RI_K, MarketSZ, "000001", 0, 10)
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
	reply, err := tdx.GetIndexBars(proto.KLINE_TYPE_RI_K, MarketSH, "000001", 0, 10)
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
	reply, err := tdx.GetMinuteTimeData(MarketSZ, "159607")
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
	reply, err := tdx.GetHistoryMinuteTimeData(20220511, MarketSZ, "159607")
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
	reply, err := tdx.GetTransactionData(MarketSZ, "159607", 0, 10)
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
	reply, err := tdx.GetHistoryTransactionData(20230922, MarketSZ, "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

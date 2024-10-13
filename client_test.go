package gotdx

import (
	"fmt"
	"github.com/bensema/gotdx/proto"
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

func Test_tdx_Connect(t *testing.T) {
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
	fmt.Println("================ GetSecurityCount ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityCount(MarketSh)
	if err != nil {
		t.Errorf("error:%s", err)
	}
	fmt.Println(reply.Count)
}

func Test_tdx_GetSecurityQuotes(t *testing.T) {
	fmt.Println("================ GetSecurityQuotes ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityQuotes([]uint8{MarketSh}, []string{"002062"})
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetSecurityList(t *testing.T) {
	fmt.Println("================ GetSecurityList ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityList(MarketSh, 0)
	if err != nil {
		t.Errorf("error:%s", err)
	}
	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetSecurityBars(t *testing.T) {
	fmt.Println("================ GetSecurityBars ================")
	// GetSecurityBars 与 GetIndexBars 使用同一个接口靠market区分
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetSecurityBars(proto.KLINE_TYPE_RI_K, 0, "000001", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetIndexBars(t *testing.T) {
	fmt.Println("================ GetIndexBars ================")
	// GetSecurityBars 与 GetIndexBars 使用同一个接口靠market区分
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetIndexBars(proto.KLINE_TYPE_RI_K, 1, "000001", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetMinuteTimeData(t *testing.T) {
	fmt.Println("================ GetMinuteTimeData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetMinuteTimeData(0, "159607")
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetHistoryMinuteTimeData(t *testing.T) {
	fmt.Println("================ GetHistoryMinuteTimeData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	//reply, err := tdx.GetHistoryMinuteTimeData(20220511, 0, "159607")
	reply, err := tdx.GetHistoryMinuteTimeData(20220511, 0, "159607")
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetTransactionData(t *testing.T) {
	fmt.Println("================ GetTransactionData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	//reply, err := tdx.GetHistoryMinuteTimeData(20220511, 0, "159607")
	reply, err := tdx.GetTransactionData(MarketSh, "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

func Test_tdx_GetHistoryTransactionData(t *testing.T) {
	fmt.Println("================ GetHistoryTransactionData ================")
	tdx := newClient()
	defer tdx.Disconnect()
	reply, err := tdx.GetHistoryTransactionData(20230922, MarketSh, "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Printf("%+v \n", obj)
	}
}

package gotdx

import (
	"fmt"
	"github.com/bensema/gotdx/proto"
	"testing"
)

var opt = &Opt{
	Host: "119.147.212.81",
	Port: 7709,
}

func prepare() *Client {
	api := NewClient(opt)
	reply, err := api.Connect()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(reply.Info)
	return api
}

func Test_tdx_Connect(t *testing.T) {
	api := NewClient(opt)
	reply, err := api.Connect()
	if err != nil {
		t.Errorf("error:%s", err)
	}
	fmt.Println(reply.Info)
	_ = api.Disconnect()

}

func Test_tdx_GetSecurityCount(t *testing.T) {
	api := prepare()
	reply, err := api.GetSecurityCount(MarketSh)
	if err != nil {
		t.Errorf("error:%s", err)
	}
	fmt.Println(reply.Count)

	_ = api.Disconnect()
}

func Test_tdx_GetSecurityQuotes(t *testing.T) {
	api := prepare()
	reply, err := api.GetSecurityQuotes([]uint8{MarketSh}, []string{"002062"})
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetSecurityList(t *testing.T) {
	api := prepare()
	reply, err := api.GetSecurityList(MarketSh, 0)
	if err != nil {
		t.Errorf("error:%s", err)
	}
	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetSecurityBars(t *testing.T) {
	// GetSecurityBars 与 GetIndexBars 使用同一个接口靠market区分
	api := prepare()
	reply, err := api.GetSecurityBars(proto.KLINE_TYPE_RI_K, 0, "000001", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetIndexBars(t *testing.T) {
	// GetSecurityBars 与 GetIndexBars 使用同一个接口靠market区分
	api := prepare()
	reply, err := api.GetIndexBars(proto.KLINE_TYPE_RI_K, 1, "000001", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetMinuteTimeData(t *testing.T) {
	api := prepare()
	reply, err := api.GetMinuteTimeData(0, "159607")
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetHistoryMinuteTimeData(t *testing.T) {
	api := prepare()
	//reply, err := api.GetHistoryMinuteTimeData(20220511, 0, "159607")
	reply, err := api.GetHistoryMinuteTimeData(20220511, 0, "159607")
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetTransactionData(t *testing.T) {
	api := prepare()
	//reply, err := api.GetHistoryMinuteTimeData(20220511, 0, "159607")
	reply, err := api.GetTransactionData(0, "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

func Test_tdx_GetHistoryTransactionData(t *testing.T) {
	api := prepare()
	//reply, err := api.GetHistoryMinuteTimeData(20220511, 0, "159607")
	reply, err := api.GetHistoryTransactionData(20220511, 0, "159607", 0, 10)
	if err != nil {
		t.Errorf("error:%s", err)
	}

	for _, obj := range reply.List {
		fmt.Println(obj)
	}

	_ = api.Disconnect()

}

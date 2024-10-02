package gotdx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"github.com/bensema/gotdx/proto"
	"io"
	"log"
	"net"
	"sync"
)

func New(opts ...Option) *Client {
	client := &Client{}

	client.opt = applyOptions(opts...)
	client.sending = make(chan bool, 1)
	client.complete = make(chan bool, 1)

	return client
}

type Client struct {
	conn     net.Conn
	opt      *Options
	complete chan bool
	sending  chan bool
	mu       sync.Mutex
}

func (client *Client) connect() error {
	conn, err := net.Dial("tcp", client.opt.TCPAddress)
	if err != nil {
		return err
	}
	client.conn = conn
	return err
}

func (client *Client) do(msg proto.Msg) error {
	sendData, err := msg.Serialize()
	if err != nil {
		return err
	}

	retryTimes := 0

	for {
		n, err := client.conn.Write(sendData)
		if n < len(sendData) {
			retryTimes++
			if retryTimes <= client.opt.MaxRetryTimes {
				log.Printf("第%d次重试\n", retryTimes)
			} else {
				return err
			}
		} else {
			if err != nil {
				return err
			}
			break
		}
	}

	headerBytes := make([]byte, proto.MessageHeaderBytes)
	_, err = io.ReadFull(client.conn, headerBytes)
	if err != nil {
		return err
	}

	headerBuf := bytes.NewReader(headerBytes)
	var header proto.RespHeader
	if err := binary.Read(headerBuf, binary.LittleEndian, &header); err != nil {
		return err
	}

	if header.ZipSize > proto.MessageMaxBytes {
		log.Printf("msgData has bytes(%d) beyond max %d\n", header.ZipSize, proto.MessageMaxBytes)
		return ErrBadData
	}

	msgData := make([]byte, header.ZipSize)
	_, err = io.ReadFull(client.conn, msgData)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	if header.ZipSize != header.UnZipSize {
		b := bytes.NewReader(msgData)
		r, _ := zlib.NewReader(b)
		io.Copy(&out, r)
		err = msg.UnSerialize(&header, out.Bytes())
	} else {
		err = msg.UnSerialize(&header, msgData)
	}

	return err
}

// Connect 连接券商行情服务器
func (client *Client) Connect() (*proto.Hello1Reply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	err := client.connect()
	if err != nil {
		return nil, err
	}
	obj := proto.NewHello1()
	err = client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// Disconnect 断开服务器
func (client *Client) Disconnect() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.conn.Close()
}

// GetSecurityCount 获取指定市场内的证券数目
func (client *Client) GetSecurityCount(market uint16) (*proto.GetSecurityCountReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetSecurityCount()
	obj.SetParams(&proto.GetSecurityCountRequest{
		Market: market,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetSecurityQuotes 获取盘口五档报价
func (client *Client) GetSecurityQuotes(markets []uint8, codes []string) (*proto.GetSecurityQuotesReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if len(markets) != len(codes) {
		return nil, errors.New("market code count error")
	}
	obj := proto.NewGetSecurityQuotes()
	var params []proto.Stock
	for i, market := range markets {
		params = append(params, proto.Stock{
			Market: market,
			Code:   codes[i],
		})
	}
	obj.SetParams(&proto.GetSecurityQuotesRequest{StockList: params})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetSecurityList 获取市场内指定范围内的所有证券代码
func (client *Client) GetSecurityList(market uint8, start uint16) (*proto.GetSecurityListReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetSecurityList()
	_market := uint16(market)
	obj.SetParams(&proto.GetSecurityListRequest{Market: _market, Start: start})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetSecurityBars 获取股票K线
func (client *Client) GetSecurityBars(category uint16, market uint8, code string, start uint16, count uint16) (*proto.GetSecurityBarsReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetSecurityBars()
	_code := [6]byte{}
	_market := uint16(market)
	copy(_code[:], code)
	obj.SetParams(&proto.GetSecurityBarsRequest{
		Market:   _market,
		Code:     _code,
		Category: category,
		Start:    start,
		Count:    count,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetIndexBars 获取指数K线
func (client *Client) GetIndexBars(category uint16, market uint8, code string, start uint16, count uint16) (*proto.GetIndexBarsReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetIndexBars()
	_code := [6]byte{}
	_market := uint16(market)
	copy(_code[:], code)
	obj.SetParams(&proto.GetIndexBarsRequest{
		Market:   _market,
		Code:     _code,
		Category: category,
		Start:    start,
		Count:    count,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetMinuteTimeData 获取分时图数据
func (client *Client) GetMinuteTimeData(market uint8, code string) (*proto.GetMinuteTimeDataReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetMinuteTimeData()
	_code := [6]byte{}
	_market := uint16(market)
	copy(_code[:], code)
	obj.SetParams(&proto.GetMinuteTimeDataRequest{
		Market: _market,
		Code:   _code,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetHistoryMinuteTimeData 获取历史分时图数据
func (client *Client) GetHistoryMinuteTimeData(date uint32, market uint8, code string) (*proto.GetHistoryMinuteTimeDataReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetHistoryMinuteTimeData()
	_code := [6]byte{}
	copy(_code[:], code)
	obj.SetParams(&proto.GetHistoryMinuteTimeDataRequest{
		Date:   date,
		Market: market,
		Code:   _code,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetTransactionData 获取分时成交
func (client *Client) GetTransactionData(market uint8, code string, start uint16, count uint16) (*proto.GetTransactionDataReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetTransactionData()
	_code := [6]byte{}
	_market := uint16(market)
	copy(_code[:], code)
	obj.SetParams(&proto.GetTransactionDataRequest{
		Market: _market,
		Code:   _code,
		Start:  start,
		Count:  count,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// GetHistoryTransactionData 获取历史分时成交
func (client *Client) GetHistoryTransactionData(date uint32, market uint8, code string, start uint16, count uint16) (*proto.GetHistoryTransactionDataReply, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewGetHistoryTransactionData()
	_code := [6]byte{}
	_market := uint16(market)
	copy(_code[:], code)
	obj.SetParams(&proto.GetHistoryTransactionDataRequest{
		Date:   date,
		Market: _market,
		Code:   _code,
		Start:  start,
		Count:  count,
	})
	err := client.do(obj)
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

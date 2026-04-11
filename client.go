package gotdx

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/bensema/gotdx/proto"
)

type clientMode uint8

const (
	clientModeMain clientMode = iota
	clientModeEx
	clientModeMacMain
	clientModeMacEx
)

func New(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeMain)
}

func NewEx(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeEx)
}

func NewMAC(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeMacMain)
}

func NewMACEx(opts ...Option) *Client {
	return newClientWithOptions(applyOptions(opts...), clientModeMacEx)
}

type Client struct {
	conn     net.Conn
	opt      *Options
	complete chan bool
	sending  chan bool
	mu       sync.Mutex
	mode     clientMode
	main     *Client
	ex       *Client
}

func (client *Client) CurrentAddress() string {
	if client == nil || client.opt == nil {
		return ""
	}
	return client.opt.TCPAddress
}

// ProbeHosts probes the client's configured address list and returns the
// results sorted by reachability and latency.
func (client *Client) ProbeHosts() []HostProbeResult {
	if client == nil || client.opt == nil {
		return nil
	}
	return ProbeAddresses(client.addresses(), client.timeout())
}

// FastestHost returns the fastest reachable configured address.
func (client *Client) FastestHost() (HostProbeResult, error) {
	if client == nil || client.opt == nil {
		return HostProbeResult{}, ErrNoReachableHosts
	}
	return FastestAddress(client.addresses(), client.timeout())
}

func newClientWithOptions(opt *Options, mode clientMode) *Client {
	client := &Client{
		opt:      cloneOptions(opt),
		sending:  make(chan bool, 1),
		complete: make(chan bool, 1),
		mode:     mode,
	}

	if mode == clientModeEx {
		client.opt.TCPAddress = client.opt.ExTCPAddress
		client.opt.TCPAddressPool = append([]string(nil), client.opt.ExTCPAddressPool...)
	}
	if mode == clientModeMacMain {
		client.opt.TCPAddress = client.opt.MacTCPAddress
		client.opt.TCPAddressPool = append([]string(nil), client.opt.MacTCPAddressPool...)
	}
	if mode == clientModeMacEx {
		client.opt.TCPAddress = client.opt.MacExTCPAddress
		client.opt.TCPAddressPool = append([]string(nil), client.opt.MacExTCPAddressPool...)
	}

	return client
}

func cloneOptions(opt *Options) *Options {
	if opt == nil {
		return defaultOptions()
	}
	clone := *opt
	clone.TCPAddressPool = append([]string(nil), opt.TCPAddressPool...)
	clone.ExTCPAddressPool = append([]string(nil), opt.ExTCPAddressPool...)
	clone.MacTCPAddressPool = append([]string(nil), opt.MacTCPAddressPool...)
	clone.MacExTCPAddressPool = append([]string(nil), opt.MacExTCPAddressPool...)
	return &clone
}

func (client *Client) connect() error {
	addresses := client.connectionOrder()
	if len(addresses) == 0 {
		return errors.New("no tcp address configured")
	}

	var lastErr error
	for _, address := range addresses {
		if err := client.connectToAddress(address); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New("no available tcp address")
	}
	return lastErr
}

func (client *Client) connectToAddress(address string) error {
	conn, err := net.DialTimeout("tcp", address, client.timeout())
	if err != nil {
		return err
	}
	client.conn = conn
	client.opt.TCPAddress = address
	return nil
}

func (client *Client) addresses() []string {
	addresses := make([]string, 0, 1+len(client.opt.TCPAddressPool))
	if client.opt.TCPAddress != "" {
		addresses = append(addresses, client.opt.TCPAddress)
	}
	addresses = append(addresses, client.opt.TCPAddressPool...)
	return addresses
}

func (client *Client) connectionOrder() []string {
	if client == nil || client.opt == nil {
		return nil
	}
	addresses := client.addresses()
	if len(addresses) == 0 {
		return nil
	}
	if !client.opt.AutoSelectFastest {
		return addresses
	}

	results := ProbeAddresses(addresses, client.timeout())
	if len(results) == 0 {
		return addresses
	}

	ordered := make([]string, 0, len(addresses))
	seen := make(map[string]struct{}, len(addresses))
	for _, result := range results {
		if !result.Reachable {
			continue
		}
		if _, ok := seen[result.Address]; ok {
			continue
		}
		ordered = append(ordered, result.Address)
		seen[result.Address] = struct{}{}
	}
	for _, address := range addresses {
		if _, ok := seen[address]; ok {
			continue
		}
		ordered = append(ordered, address)
	}
	return ordered
}

func (client *Client) timeout() time.Duration {
	if client == nil || client.opt == nil {
		return time.Duration(_defaultTimeoutSec) * time.Second
	}
	timeout := time.Duration(client.opt.TimeoutSec) * time.Second
	if timeout <= 0 {
		return time.Duration(_defaultTimeoutSec) * time.Second
	}
	return timeout
}

func (client *Client) closeCurrentConn() {
	if client.conn != nil {
		_ = client.conn.Close()
		client.conn = nil
	}
}

func (client *Client) connectWithHandshake(handshake func() error) error {
	addresses := client.connectionOrder()
	if len(addresses) == 0 {
		return errors.New("no tcp address configured")
	}

	var lastErr error
	for _, address := range addresses {
		if err := client.connectToAddress(address); err != nil {
			lastErr = err
			continue
		}
		if err := handshake(); err == nil {
			return nil
		} else {
			lastErr = err
			client.closeCurrentConn()
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no available tcp address")
	}
	return lastErr
}

func (client *Client) do(msg proto.Msg) error {
	if client.conn == nil {
		return errors.New("connection is nil")
	}

	_ = client.conn.SetDeadline(time.Now().Add(client.timeout()))
	defer func() {
		_ = client.conn.SetDeadline(time.Time{})
	}()

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
	if client.mode == clientModeEx {
		client.mu.Lock()
		if client.main == nil {
			client.main = newClientWithOptions(client.opt, clientModeMain)
		}
		main := client.main
		client.mu.Unlock()
		return main.Connect()
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewHello1()
	err := client.connectWithHandshake(func() error {
		return client.do(obj)
	})
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// ConnectEx 连接扩展市场服务器并完成登录
func (client *Client) ConnectEx() (*proto.ExLoginReply, error) {
	if client.mode == clientModeMain {
		client.mu.Lock()
		if client.ex == nil {
			client.ex = newClientWithOptions(client.opt, clientModeEx)
		}
		ex := client.ex
		client.mu.Unlock()
		return ex.ConnectEx()
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	obj := proto.NewExLogin()
	err := client.connectWithHandshake(func() error {
		return client.do(obj)
	})
	if err != nil {
		return nil, err
	}
	return obj.Reply(), err
}

// Disconnect 断开服务器
func (client *Client) Disconnect() error {
	client.mu.Lock()
	conn := client.conn
	client.conn = nil
	main := client.main
	ex := client.ex
	client.main = nil
	client.ex = nil
	client.mu.Unlock()

	var err error
	if conn != nil {
		err = conn.Close()
	}
	if main != nil && main != client {
		if closeErr := main.Disconnect(); err == nil {
			err = closeErr
		}
	}
	if ex != nil && ex != client && ex != main {
		if closeErr := ex.Disconnect(); err == nil {
			err = closeErr
		}
	}
	return err
}

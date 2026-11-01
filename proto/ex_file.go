package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// ExGetFileMeta 表示扩展市场远端文件元信息查询协议，返回文件大小与哈希校验值。
// 请求参数与响应结构复用通用文件元信息类型。
type ExGetFileMeta struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetFileMetaRequest
	reply      *GetFileMetaReply
}

// NewExGetFileMeta 构造文件元信息查询协议实例。
//
// 参数 `req` 指定待查询的目标文件名；传入非空对象时覆盖默认参数，传 `nil` 时使用空文件名请求。
// 返回的实例零值不可直接使用。
func NewExGetFileMeta(req *GetFileMetaRequest) *ExGetFileMeta {
	obj := &ExGetFileMeta{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(GetFileMetaRequest),
		reply:      new(GetFileMetaReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXFILEMETA
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExGetFileMeta) applyRequest(req *GetFileMetaRequest) {
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetFileMeta) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXFILEMETA, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码文件元信息并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应长度不足 38 字节时返回
// 非空错误，解码成功返回 `nil`。
func (obj *ExGetFileMeta) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 元信息头部至少需要 38 字节：大小 4 字节、保留 1 字节、哈希 32 字节、保留 1 字节。
	if len(data) < 38 {
		return fmt.Errorf("invalid ex file meta response length: %d", len(data))
	}
	// 文件大小，单位字节，`uint32`。
	obj.reply.Size = binary.LittleEndian.Uint32(data[:4])
	//保留位，语义待确认。
	obj.reply.Unknown1 = data[4]
	// 32 字节哈希校验值整体拷贝进定长数组，超出部分由数组容量自动截断。
	copy(obj.reply.HashValue[:], data[5:37])
	//保留位，语义待确认。
	obj.reply.Unknown2 = data[37]
	return nil
}

// Response 返回解码完成的文件元信息响应对象。
func (obj *ExGetFileMeta) Response() *GetFileMetaReply {
	return obj.reply
}

// ExDownloadFile 表示扩展市场远端文件分块下载协议，按起始偏移与块大小获取文件分块。
// 响应仅返回单个文件分块，上层需按起始偏移把各分块拼接为完整文件。
type ExDownloadFile struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExDownloadFileRequest
	reply      *DownloadFileReply
}

// ExDownloadFileRequest 保存文件分块下载的起始偏移、块大小与远端文件名参数。
type ExDownloadFileRequest struct {
	Start    uint32   // 分块起始偏移，单位字节，`uint32`，0 表示从文件头开始。
	Size     uint32   // 本次请求分块的字节数，`uint32`。
	Filename [40]byte // 远端文件名，GBK 定长 40 字节，不足部分补零。
}

// NewExDownloadFile 构造文件下载协议实例。
//
// 参数 `req` 指定分块起始偏移、块大小与文件名；传入非空对象时覆盖默认参数，传 `nil` 时使用空
// 请求。返回的实例零值不可直接使用。
func NewExDownloadFile(req *ExDownloadFileRequest) *ExDownloadFile {
	obj := &ExDownloadFile{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExDownloadFileRequest),
		reply:      new(DownloadFileReply),
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = KMSG_EXFILEDOWNLOAD
	if req != nil {
		obj.applyRequest(req)
	}
	return obj
}

func (obj *ExDownloadFile) applyRequest(req *ExDownloadFileRequest) {
	obj.request = req
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExDownloadFile) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(KMSG_EXFILEDOWNLOAD, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码文件分块并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应缺少 4 字节块大小头部时
// 返回非空错误，解码成功返回 `nil`。
func (obj *ExDownloadFile) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 至少需要 4 字节块大小头部，`uint32`。
	if len(data) < 4 {
		return fmt.Errorf("invalid ex download file response length: %d", len(data))
	}
	obj.reply.Size = binary.LittleEndian.Uint32(data[:4])
	// 4 字节头部之后全部为文件分块数据；拷贝一份独立切片，避免复用服务端返回的缓冲区。
	obj.reply.Data = append([]byte(nil), data[4:]...)
	return nil
}

// Response 返回解码完成的文件下载响应对象。
func (obj *ExDownloadFile) Response() *DownloadFileReply {
	return obj.reply
}

// ExGetTableChunk 是扩展市场表格数据分块请求的通用基类，封装公共请求编码与响应解析，
// 由表格与明细表两个派生协议嵌入复用。
type ExGetTableChunk struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetTableChunkRequest
	reply      *ExGetTableChunkReply
	method     uint16
	mode       uint8
}

// ExGetTableChunkRequest 保存表格分块查询的起始偏移、令牌与查询模式参数。
type ExGetTableChunkRequest struct {
	Start    uint32   // 分块起始偏移，`uint32`，0 表示从第一条记录开始。
	Zero     uint32   // 保留置零字段，发送时固定为 0，语义待确认。
	Token    [16]byte // 16 字节固定协议令牌，由基类构造器写入，两个表格协议共用同一份值。
	Reserved [85]byte // 85 字节保留填充区，发送时置零，语义待确认。
	Mode     uint8    // 表格查询模式：汇总表为 1、明细表为 0，由派生协议固定传入。
	Pad      [16]byte // 16 字节协议填充区，发送时置零，语义待确认。
}

// ExGetTableChunkReply 保存解码完成的表格分块响应结果。
type ExGetTableChunkReply struct {
	Start   uint32 // 服务端回显的本块起始偏移，`uint32`。
	Count   uint32 // 本块返回的表格记录条数，`uint32`。
	Content string // 表格文本内容，GBK 解码。
}

// newExGetTableChunk 构造表格分块基类实例。
//
// 参数 `method` 为派生协议使用的消息号；`mode` 为表格查询模式（汇总表 1、明细表 0）。
func newExGetTableChunk(method uint16, mode uint8) *ExGetTableChunk {
	obj := &ExGetTableChunk{
		reqHeader:  new(ReqHeader),
		respHeader: new(RespHeader),
		request:    new(ExGetTableChunkRequest),
		reply:      new(ExGetTableChunkReply),
		method:     method,
		mode:       mode,
	}
	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x01
	obj.reqHeader.Method = method
	// 固定 16 字节协议令牌，两个表格协议共用同一份值。
	copy(obj.request.Token[:], []byte{0x00, 0x78, 0x1f, 0x0e, 0x6a, 0x37, 0x44, 0x7b, 0x50, 0x2b, 0x7c, 0x0d, 0x01, 0x40, 0x4c, 0x0a})
	obj.request.Mode = mode
	return obj
}

// ExGetTable 表示扩展市场表格分类列表等汇总数据的分块查询协议，嵌入基类复用公共逻辑。
type ExGetTable struct {
	*ExGetTableChunk
}

// NewExGetTable 构造表格分块查询实例。
//
// 参数 `start` 指定本块起始偏移，0 表示从第一条记录开始；返回的实例零值不可直接使用。
func NewExGetTable(start uint32) *ExGetTable {
	obj := &ExGetTable{ExGetTableChunk: newExGetTableChunk(KMSG_EXTABLE, 1)}
	obj.applyStart(start)
	return obj
}

// ExGetTableDetail 表示扩展市场表格明细分块查询协议，嵌入基类复用公共逻辑。
type ExGetTableDetail struct {
	*ExGetTableChunk
}

// NewExGetTableDetail 构造表格明细分块查询实例。
//
// 参数 `start` 指定本块起始偏移，0 表示从第一条记录开始；返回的实例零值不可直接使用。
func NewExGetTableDetail(start uint32) *ExGetTableDetail {
	obj := &ExGetTableDetail{ExGetTableChunk: newExGetTableChunk(KMSG_EXTABLEDETAIL, 0)}
	obj.applyStart(start)
	return obj
}

func (obj *ExGetTableChunk) applyStart(start uint32) {
	obj.request.Start = start
	obj.request.Mode = obj.mode
}

// BuildRequest 把当前请求参数编码为可发送的扩展协议字节流。
//
// 返回值第一项为编码后的请求字节流；参数序列化失败时第二项返回非空错误。
func (obj *ExGetTableChunk) BuildRequest() ([]byte, error) {
	payload := new(bytes.Buffer)
	// 请求体定长序列化，字段顺序与服务端约定一致。
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return buildExRequest(obj.method, payload.Bytes())
}

// ParseResponse 从服务端响应字节流解码表格分块文本并填充响应。
//
// 参数 `header` 为协议响应头，`data` 为服务端返回的原始响应字节流；响应固定头部不足 169 字节时
// 返回非空错误，解码成功返回 `nil`。
func (obj *ExGetTableChunk) ParseResponse(header *RespHeader, data []byte) error {
	obj.respHeader = header
	// 固定头部至少需要 169 字节，之后才是表格文本区。
	if len(data) < 169 {
		return fmt.Errorf("invalid ex table response length: %d", len(data))
	}
	// 起始偏移位于 35 字节起 4 字节`uint32`，返回条数位于 161 字节起 4 字节`uint32`。
	obj.reply.Start = binary.LittleEndian.Uint32(data[35:39])
	obj.reply.Count = binary.LittleEndian.Uint32(data[161:165])
	// 169 字节之后为 GBK 编码的表格文本，整体解码为字符串。
	obj.reply.Content = Utf8ToGbk(data[169:])
	return nil
}

// Response 返回解码完成的表格分块响应对象。
func (obj *ExGetTableChunk) Response() *ExGetTableChunkReply {
	return obj.reply
}

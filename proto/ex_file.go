package proto

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type ExGetFileMeta struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetFileMetaRequest
	reply      *GetFileMetaReply
}

func NewExGetFileMeta() *ExGetFileMeta {
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
	return obj
}

func (obj *ExGetFileMeta) SetParams(req *GetFileMetaRequest) {
	obj.request = req
}

func (obj *ExGetFileMeta) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXFILEMETA, payload.Bytes())
}

func (obj *ExGetFileMeta) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 38 {
		return fmt.Errorf("invalid ex file meta response length: %d", len(data))
	}
	obj.reply.Size = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Unknown1 = data[4]
	copy(obj.reply.HashValue[:], data[5:37])
	obj.reply.Unknown2 = data[37]
	return nil
}

func (obj *ExGetFileMeta) Reply() *GetFileMetaReply {
	return obj.reply
}

type ExDownloadFile struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExDownloadFileRequest
	reply      *DownloadFileReply
}

type ExDownloadFileRequest struct {
	Start    uint32
	Size     uint32
	Filename [40]byte
}

func NewExDownloadFile() *ExDownloadFile {
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
	return obj
}

func (obj *ExDownloadFile) SetParams(req *ExDownloadFileRequest) {
	obj.request = req
}

func (obj *ExDownloadFile) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(KMSG_EXFILEDOWNLOAD, payload.Bytes())
}

func (obj *ExDownloadFile) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 4 {
		return fmt.Errorf("invalid ex download file response length: %d", len(data))
	}
	obj.reply.Size = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Data = append([]byte(nil), data[4:]...)
	return nil
}

func (obj *ExDownloadFile) Reply() *DownloadFileReply {
	return obj.reply
}

type ExGetTableChunk struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *ExGetTableChunkRequest
	reply      *ExGetTableChunkReply
	method     uint16
	mode       uint8
}

type ExGetTableChunkRequest struct {
	Start    uint32
	Zero     uint32
	Token    [16]byte
	Reserved [85]byte
	Mode     uint8
	Pad      [16]byte
}

type ExGetTableChunkReply struct {
	Start   uint32
	Count   uint32
	Content string
}

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
	copy(obj.request.Token[:], []byte{0x00, 0x78, 0x1f, 0x0e, 0x6a, 0x37, 0x44, 0x7b, 0x50, 0x2b, 0x7c, 0x0d, 0x01, 0x40, 0x4c, 0x0a})
	obj.request.Mode = mode
	return obj
}

type ExGetTable struct {
	*ExGetTableChunk
}

func NewExGetTable() *ExGetTable {
	return &ExGetTable{ExGetTableChunk: newExGetTableChunk(KMSG_EXTABLE, 1)}
}

type ExGetTableDetail struct {
	*ExGetTableChunk
}

func NewExGetTableDetail() *ExGetTableDetail {
	return &ExGetTableDetail{ExGetTableChunk: newExGetTableChunk(KMSG_EXTABLEDETAIL, 0)}
}

func (obj *ExGetTableChunk) SetParams(start uint32) {
	obj.request.Start = start
	obj.request.Mode = obj.mode
}

func (obj *ExGetTableChunk) Serialize() ([]byte, error) {
	payload := new(bytes.Buffer)
	if err := binary.Write(payload, binary.LittleEndian, obj.request); err != nil {
		return nil, err
	}
	return serializeExRequest(obj.method, payload.Bytes())
}

func (obj *ExGetTableChunk) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	if len(data) < 169 {
		return fmt.Errorf("invalid ex table response length: %d", len(data))
	}
	obj.reply.Start = binary.LittleEndian.Uint32(data[35:39])
	obj.reply.Count = binary.LittleEndian.Uint32(data[161:165])
	obj.reply.Content = Utf8ToGbk(data[169:])
	return nil
}

func (obj *ExGetTableChunk) Reply() *ExGetTableChunkReply {
	return obj.reply
}

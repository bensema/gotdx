package proto

import (
	"bytes"
	"encoding/binary"
)

type GetFileMeta struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *GetFileMetaRequest
	reply      *GetFileMetaReply
}

type GetFileMetaRequest struct {
	Filename [40]byte
}

type GetFileMetaReply struct {
	Size      uint32
	Unknown1  byte
	HashValue [32]byte
	Unknown2  byte
}

func NewGetFileMeta() *GetFileMeta {
	obj := new(GetFileMeta)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(GetFileMetaRequest)
	obj.reply = new(GetFileMetaReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_BLOCKINFOMETA
	return obj
}

func (obj *GetFileMeta) SetParams(req *GetFileMetaRequest) {
	obj.request = req
}

func (obj *GetFileMeta) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 42
	obj.reqHeader.PkgLen2 = 42

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *GetFileMeta) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	obj.reply.Size = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Unknown1 = data[4]
	copy(obj.reply.HashValue[:], data[5:37])
	obj.reply.Unknown2 = data[37]
	return nil
}

func (obj *GetFileMeta) Reply() *GetFileMetaReply {
	return obj.reply
}

type DownloadFile struct {
	reqHeader  *ReqHeader
	respHeader *RespHeader
	request    *DownloadFileRequest
	reply      *DownloadFileReply
}

type DownloadFileRequest struct {
	Start    uint32
	Size     uint32
	Filename [300]byte
}

type DownloadFileReply struct {
	Size uint32
	Data []byte
}

func NewDownloadFile() *DownloadFile {
	obj := new(DownloadFile)
	obj.reqHeader = new(ReqHeader)
	obj.respHeader = new(RespHeader)
	obj.request = new(DownloadFileRequest)
	obj.reply = new(DownloadFileReply)

	obj.reqHeader.Zip = 0x0c
	obj.reqHeader.SeqID = seqID()
	obj.reqHeader.PacketType = 0x00
	obj.reqHeader.Method = KMSG_BLOCKINFO
	return obj
}

func (obj *DownloadFile) SetParams(req *DownloadFileRequest) {
	obj.request = req
}

func (obj *DownloadFile) Serialize() ([]byte, error) {
	obj.reqHeader.PkgLen1 = 310
	obj.reqHeader.PkgLen2 = 310

	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, obj.reqHeader)
	err = binary.Write(buf, binary.LittleEndian, obj.request)
	return buf.Bytes(), err
}

func (obj *DownloadFile) UnSerialize(header interface{}, data []byte) error {
	obj.respHeader = header.(*RespHeader)
	obj.reply.Size = binary.LittleEndian.Uint32(data[:4])
	obj.reply.Data = append([]byte(nil), data[4:]...)
	return nil
}

func (obj *DownloadFile) Reply() *DownloadFileReply {
	return obj.reply
}

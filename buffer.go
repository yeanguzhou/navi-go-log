package navi_go_log

import (
	"bytes"
	"io"
	"sync"
)

var bytesBufPool = sync.Pool{
	// New is called when a new instance is needed
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func GetBytesBuffer() *bytes.Buffer {
	buf := bytesBufPool.Get().(*bytes.Buffer)
	return buf
}

func PutBytesBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bytesBufPool.Put(buf)
}

var logRecodeHandlePool = sync.Pool{
	// New is called when a new instance is needed
	New: func() interface{} {
		return &logRecodeHandle{}
	},
}

type logRecodeHandle struct {
	Buf *bytes.Buffer
	W   io.Writer
}

func GetLogRecodeHandle() *logRecodeHandle {
	lgr := logRecodeHandlePool.Get().(*logRecodeHandle)
	return lgr
}

func PutLogRecodeHandle(lgr *logRecodeHandle) {
	lgr.W = nil
	logRecodeHandlePool.Put(lgr)
}

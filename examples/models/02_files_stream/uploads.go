package files_streams

import (
	"io"
	"mime/multipart"
	"os"
)

// StandardFileTypes — авто-детект стандартных типов
// @oa:description "Upload request with standard file/stream types"
type StandardFileTypes struct {
	// @oa:description "Avatar as *os.File"
	// @oa:example "avatar.png"
	Avatar *os.File

	// @oa:description "Payload as io.Reader"
	// @oa:example "payload-content"
	Payload io.Reader

	// @oa:description "Thumbnail as io.ReadCloser"
	// @oa:example "thumb.png"
	Thumbnail io.ReadCloser

	// @oa:description "Data as base64 []byte"
	// @oa:example "SGVsbG8gV29ybGQ="
	Data []byte

	// @oa:description "Multipart file"
	// @oa:example "attachment.zip"
	Attachment multipart.File
}

// CustomWithAnnotations — кастомные типы с явными аннотациями
// @oa:description "Upload with custom reader types"
type CustomWithAnnotations struct {
	// @oa:file
	// @oa:description "Custom file reader"
	// @oa:example "custom-avatar.png"
	Avatar *CustomReader

	// @oa:stream
	// @oa:description "Raw stream from custom type"
	// @oa:example "custom-stream-data"
	Payload *CustomStream

	// @oa:byte
	// @oa:description "Data as byte array"
	// @oa:example "SGVsbG8gV29ybGQ="
	Data []byte
}

// MixedRequest — комбинация файлов и обычных полей
// @oa:description "Mixed upload request"
type MixedRequest struct {
	// @oa:file
	// @oa:description "Main file"
	// @oa:example "main-file.txt"
	File *os.File

	// @oa:description "User ID from form"
	// @oa:example "user_123"
	UserID string

	// @oa:description "Optional description"
	// @oa:example "A nice description"
	Description string

	// @oa:description "Tags as comma-separated"
	// @oa:example ["tag1", "tag2"]
	Tags []string

	// @evl:validate required
	// @evl:validate min:1
	// @oa:description "At least one category"
	// @oa:example ["category1", "category2"]
	Categories []string
}

// CustomReader — кастомный тип, реализующий io.Reader
type CustomReader struct {
	data []byte
	pos  int
}

func (c *CustomReader) Read(p []byte) (int, error) {
	if c.pos >= len(c.data) {
		return 0, io.EOF
	}
	n := copy(p, c.data[c.pos:])
	c.pos += n
	return n, nil
}

// CustomStream — кастомный тип для потока данных
type CustomStream struct {
	data []byte
	pos  int
}

func (c *CustomStream) Read(p []byte) (int, error) {
	if c.pos >= len(c.data) {
		return 0, io.EOF
	}
	n := copy(p, c.data[c.pos:])
	c.pos += n
	return n, nil
}

// CustomBuffer — кастомный тип для буфера данных
type CustomBuffer struct {
	data []byte
}

func (c *CustomBuffer) Bytes() []byte {
	return c.data
}

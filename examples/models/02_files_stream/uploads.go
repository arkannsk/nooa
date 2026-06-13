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
	Avatar *os.File

	// @oa:description "Payload as io.Reader"
	Payload io.Reader

	// @oa:description "Thumbnail as io.ReadCloser"
	Thumbnail io.ReadCloser

	// @oa:description "Data as base64 []byte"
	Data []byte

	// @oa:description "Multipart file"
	Attachment multipart.File
}

// CustomWithAnnotations — кастомные типы с явными аннотациями
// @oa:description "Upload with custom reader types"
type CustomWithAnnotations struct {
	// @oa:file
	// @oa:description "Custom file reader"
	Avatar *CustomReader

	// @oa:stream
	// @oa:description "Raw stream from custom type"
	Payload *CustomStream

	// @oa:format:byte
	// @oa:description "Base64 data from custom type"
	Data *CustomBuffer
}

// MixedRequest — комбинация файлов и обычных полей
// @oa:description "Mixed upload request"
type MixedRequest struct {
	// @oa:file
	// @oa:description "Main file"
	File *os.File

	// @oa:description "User ID from form"
	UserID string

	// @oa:description "Optional description"
	Description *string

	// @oa:description "Tags as comma-separated"
	Tags []string

	// @evl:validate required
	// @evl:validate min:1
	// @oa:description "At least one category"
	Categories []string
}

// CustomReader — кастомный тип, реализующий io.Reader
// (не должен авто-детектиться без @oa:file)
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

type CustomStream struct { /* ... */
}
type CustomBuffer struct { /* ... */
}

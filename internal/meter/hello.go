package meter

import (
	"encoding/binary"
	"errors"
	"strings"
)

var (
	// ErrIncomplete báo ClientHello chưa đủ byte; cần gom thêm dữ liệu rồi đọc lại.
	ErrIncomplete = errors.New("incomplete client hello")
	// ErrMalformed báo dữ liệu không phải ClientHello hợp lệ.
	ErrMalformed = errors.New("malformed client hello")
	// ErrNoName báo ClientHello hợp lệ nhưng không có server_name (ví dụ ECH hoặc truy cập thẳng IP).
	ErrNoName = errors.New("no server name")
)

const (
	recordHeader      = 5
	maxRecord         = 1 << 14
	handshakeHeader   = 4
	handshakeHello    = 1
	recordHandshake   = 0x16
	extensionName     = 0
	nameTypeHost      = 0
	randomSize        = 32
	versionSize       = 2
	maxHostName       = 255
	lengthSize1       = 1
	lengthSize2       = 2
	lengthSize3       = 3
	minimumTLSVersion = 0x0301
)

// Con trỏ đọc có kiểm tra biên; mọi lần đọc vượt biên đều trả lỗi thay vì panic.
type reader struct {
	data []byte
	err  error
}

func (r *reader) take(size int) []byte {
	if r.err != nil || size < 0 || size > len(r.data) {
		r.err = ErrMalformed
		return nil
	}
	chunk := r.data[:size]
	r.data = r.data[size:]
	return chunk
}

func (r *reader) number(size int) int {
	chunk := r.take(size)
	value := 0
	for _, b := range chunk {
		value = value<<8 | int(b)
	}
	return value
}

// ServerName đọc server_name từ các byte đầu của luồng client; trả ErrIncomplete khi cần thêm dữ liệu.
func ServerName(stream []byte) (string, error) {
	if len(stream) < recordHeader {
		return "", ErrIncomplete
	}
	if stream[0] != recordHandshake || int(binary.BigEndian.Uint16(stream[1:3])) < minimumTLSVersion {
		return "", ErrMalformed
	}
	length := int(binary.BigEndian.Uint16(stream[3:5]))
	if length == 0 || length > maxRecord {
		return "", ErrMalformed
	}
	if len(stream) < recordHeader+length {
		return "", ErrIncomplete
	}
	return parseHandshake(stream[recordHeader : recordHeader+length])
}

func parseHandshake(body []byte) (string, error) {
	r := &reader{data: body}
	if r.number(lengthSize1) != handshakeHello {
		return "", ErrMalformed
	}
	size := r.number(lengthSize3)
	hello := &reader{data: r.take(size)}
	if r.err != nil {
		return "", ErrMalformed
	}
	hello.take(versionSize + randomSize)
	hello.take(hello.number(lengthSize1))
	hello.take(hello.number(lengthSize2))
	hello.take(hello.number(lengthSize1))
	if hello.err != nil {
		return "", ErrMalformed
	}
	if len(hello.data) == 0 {
		return "", ErrNoName
	}
	extensions := &reader{data: hello.take(hello.number(lengthSize2))}
	for len(extensions.data) > 0 && extensions.err == nil {
		kind := extensions.number(lengthSize2)
		content := extensions.take(extensions.number(lengthSize2))
		if extensions.err == nil && kind == extensionName {
			return parseName(content)
		}
	}
	if extensions.err != nil {
		return "", ErrMalformed
	}
	return "", ErrNoName
}

func parseName(content []byte) (string, error) {
	r := &reader{data: content}
	list := &reader{data: r.take(r.number(lengthSize2))}
	for len(list.data) > 0 && list.err == nil {
		kind := list.number(lengthSize1)
		host := list.take(list.number(lengthSize2))
		if list.err != nil {
			break
		}
		if kind == nameTypeHost {
			return cleanHost(host)
		}
	}
	return "", ErrMalformed
}

func cleanHost(host []byte) (string, error) {
	if len(host) == 0 || len(host) > maxHostName {
		return "", ErrMalformed
	}
	for _, b := range host {
		const printableLow, printableHigh = 0x21, 0x7e
		if b < printableLow || b > printableHigh {
			return "", ErrMalformed
		}
	}
	return strings.ToLower(string(host)), nil
}

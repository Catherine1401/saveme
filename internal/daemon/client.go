package daemon

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"
	"time"

	"saveme/internal/core"
)

type Client struct {
	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
}

func Dial(path string) (*Client, error) {
	const timeout = 3 * time.Second
	conn, err := dial(path, timeout)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, reader: bufio.NewReader(conn)}, nil
}

func (c *Client) Call(request core.Request) (core.Response, error) {
	const timeout, newline, limit = 5 * time.Second, '\n', 2 * 1024 * 1024
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.SetDeadline(time.Now().Add(timeout))
	if err := json.NewEncoder(c.conn).Encode(request); err != nil {
		return core.Response{}, err
	}
	line, err := c.reader.ReadBytes(newline)
	if err != nil {
		return core.Response{}, err
	}
	if len(line) > limit {
		return core.Response{}, core.ErrInvalid
	}
	var response core.Response
	err = json.Unmarshal(line, &response)
	return response, err
}

func (c *Client) Close() error { return c.conn.Close() }

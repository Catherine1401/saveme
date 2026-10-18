package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"saveme/internal/core"
)

func (s *Server) Serve(ctx context.Context) error {
	const maxConnections, one = 32, 1
	var group sync.WaitGroup
	defer group.Wait()
	go func() { <-ctx.Done(); s.listener.Close() }()
	var active atomic.Int64
	s.Engine.mu.Lock()
	s.Engine.ClientCount = func() int { return int(active.Load()) }
	s.Engine.mu.Unlock()
	var owner uint64
	slots := make(chan struct{}, maxConnections)
	for {
		conn, err := s.accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if conn == nil {
			continue
		}
		select {
		case slots <- struct{}{}:
		default:
			conn.Close()
			continue
		}
		owner++
		active.Add(one)
		group.Add(one)
		go func(id uint64) {
			defer group.Done()
			defer active.Add(-one)
			defer func() { <-slots }()
			s.serveConn(ctx, conn, id)
		}(owner)
	}
}

func decodeRequest(data []byte) (core.Request, error) {
	var request core.Request
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, err
	}
	var extra core.Request
	if err := decoder.Decode(&extra); err != io.EOF {
		return request, core.ErrInvalid
	}
	return request, nil
}

func (s *Server) serveConn(ctx context.Context, conn net.Conn, owner uint64) {
	const maxLine, bufferSize, idle, writeTimeout = 64 * 1024, 4096, 5 * time.Minute, 5 * time.Second
	defer conn.Close()
	defer s.Engine.Release(owner)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-done:
		}
	}()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, bufferSize), maxLine)
	encoder := json.NewEncoder(conn)
	for {
		conn.SetReadDeadline(time.Now().Add(idle))
		if !scanner.Scan() {
			return
		}
		request, err := decodeRequest(scanner.Bytes())
		response := core.Response{Error: core.ErrInvalid}
		if err == nil {
			response = s.Engine.Handle(request, owner)
		}
		conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		if err := encoder.Encode(response); err != nil {
			return
		}
	}
}

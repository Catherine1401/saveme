package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"saveme/internal/core"
)

type Server struct {
	Engine   *Engine
	UID      uint32
	listener *net.UnixListener
	lock     *os.File
}

func (s *Server) Listen(path string) error {
	const network, suffix, mode = "unix", ".lock", 0600
	const zero, unchangedGroup = 0, -1
	lock, err := os.OpenFile(path+suffix, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return err
	}
	s.lock = lock
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSocket == zero {
			s.Close()
			return core.ErrInvalid
		}
		if err := os.Remove(path); err != nil {
			s.Close()
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		s.Close()
		return err
	}
	listener, err := net.ListenUnix(network, &net.UnixAddr{Name: path, Net: network})
	if err != nil {
		s.Close()
		return err
	}
	s.listener = listener
	if err := os.Chmod(path, mode); err != nil {
		s.Close()
		return err
	}
	if err := os.Chown(path, int(s.UID), unchangedGroup); err != nil {
		s.Close()
		return err
	}
	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
	if s.lock != nil {
		s.lock.Close()
	}
}

func peerUID(conn *net.UnixConn) (uint32, error) {
	const zero = 0
	raw, err := conn.SyscallConn()
	if err != nil {
		return zero, err
	}
	var credentials *syscall.Ucred
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return zero, err
	}
	if controlErr != nil {
		return zero, controlErr
	}
	return credentials.Uid, nil
}

func (s *Server) Serve(ctx context.Context) error {
	const maxConnections, one = 32, 1
	var group sync.WaitGroup
	defer group.Wait()
	go func() { <-ctx.Done(); s.listener.Close() }()
	var owner uint64
	slots := make(chan struct{}, maxConnections)
	for {
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		uid, err := peerUID(conn)
		if err != nil || uid != s.UID {
			conn.Close()
			continue
		}
		select {
		case slots <- struct{}{}:
		default:
			conn.Close()
			continue
		}
		owner++
		group.Add(one)
		go func(id uint64) {
			defer group.Done()
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

package daemon

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

type Server struct {
	Engine   *Engine
	SID      string
	listener net.Listener
}

// Mở named pipe chỉ cho SYSTEM và SID được phép đọc ghi; tài khoản khác bị Windows từ chối.
func (s *Server) Listen(path string) error {
	const descriptor = "D:P(A;;GA;;;SY)(A;;GRGW;;;%s)"
	listener, err := winio.ListenPipe(path, &winio.PipeConfig{SecurityDescriptor: fmt.Sprintf(descriptor, s.SID)})
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// Quyền đã được kiểm bởi security descriptor của pipe.
func (s *Server) accept() (net.Conn, error) {
	return s.listener.Accept()
}

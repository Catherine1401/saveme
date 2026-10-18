package update

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"saveme/internal/config"
)

func Serve(ctx context.Context, cfg config.Config, engine *Engine) error {
	const maximumClients = 32
	listener, unlock, err := listen(engine.Paths.Endpoint, cfg)
	if err != nil {
		return err
	}
	defer unlock()
	defer listener.Close()
	var clients sync.WaitGroup

	slots := make(chan struct{}, maximumClients)
	go func() { <-ctx.Done(); listener.Close() }()
	defer clients.Wait()
	for {
		const literal1 = 1
		connection, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		if !authorized(connection, cfg) {
			connection.Close()
			continue
		}
		select {
		case slots <- struct{}{}:
		default:
			connection.Close()
			continue
		}
		clients.Add(literal1)
		go func() {
			defer clients.Done()
			defer func() { <-slots }()
			defer connection.Close()
			finished := make(chan struct{})
			defer close(finished)
			replied := make(chan struct{})
			go func() {
				select {
				case <-ctx.Done():
					<-replied
					connection.Close()
				case <-finished:
				}
			}()
			handleConnection(ctx, connection, engine, replied)
		}()
	}
}

func handleConnection(ctx context.Context, connection net.Conn, engine *Engine, replied chan struct{}) {
	const limit, newline, timeout = 4096, '\n', 5 * time.Second
	const open, status, check, empty = "open", "status", "check", ""
	const one = 1
	var reply sync.Once
	acknowledge := func() { reply.Do(func() { close(replied) }) }
	defer acknowledge()

	connection.SetReadDeadline(time.Now().Add(timeout))
	line, err := bufio.NewReader(io.LimitReader(connection, limit)).ReadString(newline)
	if err != nil {
		return
	}
	var request Request
	decoder := json.NewDecoder(strings.NewReader(line))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return
	}
	var trailing json.RawMessage
	if decoder.Decode(&trailing) != io.EOF {
		return
	}

	response := Response{}
	var release func()
	switch request.Operation {
	case open:
		response.Status, release = engine.Open(ctx, request.Automatic)
	case status:
		response.Status = engine.State()
	case check:
		response.Status = engine.Check(ctx, true)
	default:
		response.Error = os.ErrInvalid.Error()
	}
	if release != nil {
		defer release()
	}
	if response.Status.Fatal {
		response.Error = response.Status.LastError
	}
	connection.SetReadDeadline(time.Time{})
	connection.SetWriteDeadline(time.Now().Add(timeout))
	err = json.NewEncoder(connection).Encode(response)
	acknowledge()
	if err != nil || release == nil || response.Error != empty {
		return
	}

	buffer := make([]byte, one)
	connection.Read(buffer)
}

func Connect(paths Layout, request Request) (Response, net.Conn, error) {
	const limit = 64 * 1024
	const timeout = 5 * time.Minute
	var response Response
	connection, err := dial(paths.Endpoint)
	if err != nil {
		return response, nil, err
	}
	connection.SetDeadline(time.Now().Add(timeout))
	if err := json.NewEncoder(connection).Encode(request); err != nil {
		connection.Close()
		return response, nil, err
	}

	if err := json.NewDecoder(io.LimitReader(connection, limit)).Decode(&response); err != nil {
		connection.Close()
		return response, nil, err
	}
	connection.SetDeadline(time.Time{})
	return response, connection, nil
}

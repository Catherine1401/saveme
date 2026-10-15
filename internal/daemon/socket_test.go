package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"

	"saveme/internal/core"
)

func TestProtocolOverPipe(t *testing.T) {
	const owner, duration = 1, "12h"
	const target = 0
	f := newFixture(t)
	server, client := net.Pipe()
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := Server{Engine: f.engine}
	go s.serveConn(ctx, server, owner)
	encoder, decoder := json.NewEncoder(client), json.NewDecoder(client)
	if err := encoder.Encode(core.Request{Cmd: core.CreateRange, Entry: at(target), Duration: duration}); err != nil {
		t.Fatal(err)
	}
	var pending core.Response
	if err := decoder.Decode(&pending); err != nil || !pending.Pending {
		t.Fatal(pending, err)
	}
	if err := encoder.Encode(core.Request{Cmd: core.Confirm, Code: pending.Code}); err != nil {
		t.Fatal(err)
	}
	var active core.Response
	if err := decoder.Decode(&active); err != nil || !active.Active || active.Entries[target].Windows != nil {
		t.Fatal(active, err)
	}
}

func TestMalformedProtocol(t *testing.T) {
	const (
		textCmdStatusUnknown1 = "{\"cmd\":\"status\",\"unknown\":1}\n"
		textCmdStatus         = "{\"cmd\":\"status\"} {}\n"
		textNotJson           = "not-json\n"
	)

	const owner = 1
	f := newFixture(t)
	server, client := net.Pipe()
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := Server{Engine: f.engine}
	go s.serveConn(ctx, server, owner)
	input := []string{textCmdStatusUnknown1, textCmdStatus, textNotJson}
	reader := bufio.NewReader(client)
	for _, line := range input {
		if _, err := client.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
		var response core.Response
		if err := json.NewDecoder(reader).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.Error != core.ErrInvalid {
			t.Fatal(response)
		}
	}
}

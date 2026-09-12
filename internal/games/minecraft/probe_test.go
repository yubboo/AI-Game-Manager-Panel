package minecraft

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func TestProbeReadsMinecraftStatusProtocol(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	done := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		for i := 0; i < 2; i++ {
			n, err := readVarInt(reader)
			if err != nil {
				done <- err
				return
			}
			if n < 0 || n > 1<<20 {
				done <- fmt.Errorf("invalid request packet length %d", n)
				return
			}
			if _, err := io.CopyN(io.Discard, reader, int64(n)); err != nil {
				done <- err
				return
			}
		}
		status := `{"version":{"name":"1.21.1","protocol":767},"players":{"max":10,"online":3},"description":{"text":"AGMP test"}}`
		payload := appendVarInt(nil, 0)
		payload = appendString(payload, status)
		done <- writePacket(conn, payload)
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	got, err := Probe(ctx, "127.0.0.1", port)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Online || got.Version != "1.21.1" || got.Protocol != 767 || got.PlayersOnline != 3 || got.PlayersMax != 10 || got.Description != "AGMP test" {
		t.Fatalf("unexpected probe result: %#v", got)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

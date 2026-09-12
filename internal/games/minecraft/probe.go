package minecraft

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

func Probe(ctx context.Context, host string, port int) (ProbeResult, error) {
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	if port < 1 || port > 65535 {
		return ProbeResult{}, errors.New("Minecraft 端口无效")
	}
	start := time.Now()
	d := net.Dialer{Timeout: 3 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprint(port)))
	if err != nil {
		return ProbeResult{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(4 * time.Second))
	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	handshake := appendVarInt(nil, 0)
	handshake = appendVarInt(handshake, -1)
	handshake = appendString(handshake, host)
	var pb [2]byte
	binary.BigEndian.PutUint16(pb[:], uint16(port))
	handshake = append(handshake, pb[:]...)
	handshake = appendVarInt(handshake, 1)
	if err := writePacket(rw, handshake); err != nil {
		return ProbeResult{}, err
	}
	if err := writePacket(rw, []byte{0}); err != nil {
		return ProbeResult{}, err
	}
	if err := rw.Flush(); err != nil {
		return ProbeResult{}, err
	}
	length, err := readVarInt(rw)
	if err != nil {
		return ProbeResult{}, err
	}
	if length <= 0 || length > 2<<20 {
		return ProbeResult{}, errors.New("Minecraft status packet 长度异常")
	}
	packet := make([]byte, length)
	if _, err := io.ReadFull(rw, packet); err != nil {
		return ProbeResult{}, err
	}
	br := bufio.NewReader(strings.NewReader(string(packet)))
	id, err := readVarInt(br)
	if err != nil || id != 0 {
		return ProbeResult{}, errors.New("Minecraft status 响应类型无效")
	}
	n, err := readVarInt(br)
	if err != nil || n < 0 || n > 2<<20 {
		return ProbeResult{}, errors.New("Minecraft status JSON 长度无效")
	}
	raw := make([]byte, n)
	if _, err := io.ReadFull(br, raw); err != nil {
		return ProbeResult{}, err
	}
	var status struct {
		Version struct {
			Name     string `json:"name"`
			Protocol int    `json:"protocol"`
		} `json:"version"`
		Players struct {
			Max    int `json:"max"`
			Online int `json:"online"`
		} `json:"players"`
		Description any `json:"description"`
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		return ProbeResult{}, err
	}
	desc := ""
	switch v := status.Description.(type) {
	case string:
		desc = v
	case map[string]any:
		if t, ok := v["text"].(string); ok {
			desc = t
		}
	}
	return ProbeResult{Online: true, Version: status.Version.Name, Protocol: status.Version.Protocol, PlayersOnline: status.Players.Online, PlayersMax: status.Players.Max, Description: desc, LatencyMs: time.Since(start).Milliseconds()}, nil
}

func appendVarInt(dst []byte, value int) []byte {
	u := uint32(value)
	for {
		b := byte(u & 0x7f)
		u >>= 7
		if u != 0 {
			b |= 0x80
		}
		dst = append(dst, b)
		if u == 0 {
			return dst
		}
	}
}
func appendString(dst []byte, s string) []byte {
	dst = appendVarInt(dst, len([]byte(s)))
	return append(dst, []byte(s)...)
}
func writePacket(w io.Writer, payload []byte) error {
	header := appendVarInt(nil, len(payload))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
func readVarInt(r interface{ ReadByte() (byte, error) }) (int, error) {
	var result uint32
	for i := 0; i < 5; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= uint32(b&0x7f) << uint(7*i)
		if b&0x80 == 0 {
			return int(int32(result)), nil
		}
	}
	return 0, errors.New("VarInt too long")
}

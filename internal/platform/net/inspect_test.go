package platformnet

import (
	"net"
	"testing"
)

func TestInspectUDPDetectsBoundPort(t *testing.T) {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port
	statuses, err := InspectUDP([]int{port})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || !statuses[0].Occupied {
		t.Fatalf("expected UDP %d to be occupied: %#v", port, statuses)
	}
}

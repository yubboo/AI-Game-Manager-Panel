//go:build !windows

package platformnet

import (
	"errors"
	"net"
)

func InspectUDP(ports []int) ([]Status, error) {
	ports = UniquePorts(ports)
	result := make([]Status, 0, len(ports))
	for _, port := range ports {
		status := Status{Port: port, Owners: []Owner{}}
		udp, err := net.ListenUDP("udp4", &net.UDPAddr{Port: port})
		if err == nil {
			_ = udp.Close()
		} else {
			status.Occupied = true
			status.Owners = append(status.Owners, Owner{Port: port, Protocol: ProtocolUDP})
		}
		result = append(result, status)
	}
	return result, nil
}

func Inspect(ports []int) ([]Status, error) {
	ports = UniquePorts(ports)
	result := make([]Status, 0, len(ports))
	for _, port := range ports {
		status := Status{Port: port, Owners: []Owner{}}
		udp, udpErr := net.ListenUDP("udp4", &net.UDPAddr{Port: port})
		if udpErr == nil {
			_ = udp.Close()
		} else {
			status.Occupied = true
			status.Owners = append(status.Owners, Owner{Port: port, Protocol: ProtocolUDP})
		}
		tcp, tcpErr := net.Listen("tcp4", net.JoinHostPort("0.0.0.0", fmtPort(port)))
		if tcpErr == nil {
			_ = tcp.Close()
		} else {
			status.Occupied = true
			status.Owners = append(status.Owners, Owner{Port: port, Protocol: ProtocolTCP})
		}
		result = append(result, status)
	}
	return result, nil
}

func fmtPort(port int) string {
	if port == 0 {
		return "0"
	}
	buf := make([]byte, 0, 5)
	for port > 0 {
		buf = append([]byte{byte('0' + port%10)}, buf...)
		port /= 10
	}
	return string(buf)
}

func ProcessPath(int) (string, error) {
	return "", errors.New("process path lookup is only supported on Windows")
}
func IsProcessAlive(int) bool { return false }
func TerminatePID(int) error {
	return errors.New("process termination by port owner is only supported on Windows")
}

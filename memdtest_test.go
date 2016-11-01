package memdtest

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestLaunching(t *testing.T) {
	memd, err := NewServer(true, nil)
	if err != nil {
		t.Errorf("Failed to start memcached: %s", err)
	}
	defer memd.Stop()
}

func TestUnixSock(t *testing.T) {
	memd, err := NewServer(true, Config{"unixsocket": "/tmp/______.sock"})
	if err != nil {
		t.Errorf("Failed to start memcached: %s", err)
	}
	defer memd.Stop()

	conn, err := net.Dial("unix", "/tmp/______.sock")
	if err != nil {
		t.Fatalf("Failed to connect to memcached via unix domain socket: %s", err)
	}

	fmt.Fprintf(conn, "version\r\n\r\n")
	status, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil || !strings.HasPrefix(status, "VERSION") {
		t.Fatalf("Failed to connect to memcached via unix domain socket: %s", err)
	}
}

func TestPort(t *testing.T) {
	memd, err := NewServer(true, Config{"port": "11311"})
	if err != nil {
		t.Errorf("Failed to start memcached: %s", err)
	}
	defer memd.Stop()

	conn, err := net.Dial("tcp", "localhost:11311")
	if err != nil {
		t.Fatalf("Failed to connect to memcached via tcp port: %s", err)
	}

	fmt.Fprintf(conn, "version\r\n\r\n")
	status, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil || !strings.HasPrefix(status, "VERSION") {
		t.Fatalf("Failed to connect to memcached via tcp port: %s", err)
	}
}

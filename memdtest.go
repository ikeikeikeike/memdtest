package memdtest

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Server struct {
	Config  Config
	cmd     *exec.Cmd
	TempDir string
}

type Config map[string]string

func (config Config) write(wc io.Writer) error {
	for key, value := range config {
		if _, err := fmt.Fprintf(wc, "%s %s\n", key, value); err != nil {
			return err
		}
	}
	return nil
}

func NewServer(autostart bool, config Config) (*Server, error) {
	srv := new(Server)

	if config == nil {
		config = Config{}
	}

	dir, err := ioutil.TempDir("", "memdtest")
	if err != nil {
		return nil, err
	}
	srv.TempDir = dir

	if config["unixsocket"] == "" && config["port"] == "" {
		config["unixsocket"] = filepath.Join(srv.TempDir, "memcached.sock")
	}
	srv.Config = config

	if autostart {
		if err := srv.Start(); err != nil {
			return nil, err
		}
	}

	return srv, nil
}

func (srv *Server) Start() error {

	path, err := exec.LookPath("memcached")
	if err != nil {
		return err
	}

	var args []string
	if srv.Config["unixsocket"] != "" {
		args = []string{"-s", srv.Config["unixsocket"]}
	} else {
		args = []string{"-p", srv.Config["port"]}
	}

	cmd := exec.Command(path, args...)
	srv.cmd = cmd

	// start
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "Failed to launch memcached")
	}

	checktimeout := time.NewTimer(20 * time.Second)
	checktick := time.NewTicker(time.Second)
	defer checktimeout.Stop()
	defer checktick.Stop()

	for cmd.Process == nil {
		select {
		case <-checktimeout.C:
			if proc := cmd.Process; proc != nil {
				proc.Kill()
			}
			return errors.New("Failed to launch memcached (timeout)")
		case <-checktick.C:
		}
	}

	conntimeout := time.NewTimer(30 * time.Second)
	conntick := time.NewTicker(time.Second)
	defer conntimeout.Stop()
	defer conntick.Stop()

	pr, addr := "unix", srv.Config["unixsocket"]
	if srv.Config["port"] != "" {
		pr, addr = "tcp", ":"+srv.Config["port"]
	}

	for {
		select {
		case <-conntimeout.C:
			if proc := cmd.Process; proc != nil {
				proc.Kill()
			}
			return errors.New("timeout, couldnt connect to memcached")
		case <-conntick.C:
			conn, err := net.Dial(pr, addr)
			if err != nil {
				continue
			}

			fmt.Fprintf(conn, "version\r\n\r\n")
			status, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil || !strings.HasPrefix(status, "VERSION") {
				continue
			}

			srv.cmd = cmd
			return nil
		}
	}
}

func (srv *Server) Stop() error {
	defer os.RemoveAll(srv.TempDir)
	if err := srv.killAndWait(); err != nil {
		return err
	}
	return nil
}

func (srv *Server) killAndWait() error {
	if err := srv.cmd.Process.Kill(); err != nil {
		return err
	}
	if _, err := srv.cmd.Process.Wait(); err != nil {
		return err
	}
	return nil
}

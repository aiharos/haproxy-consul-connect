package haproxy_cmd

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/criteo/haproxy-consul-connect/haproxy/dataplane"
	"github.com/criteo/haproxy-consul-connect/lib"
)

type Config struct {
	HAProxyPath             string
	HAProxyConfigPath       string
	DataplanePath           string
	DataplaneTransactionDir string
	DataplaneSock           string
	DataplaneUser           string
	DataplanePass           string
}

func Start(sd *lib.Shutdown, cfg Config) (*dataplane.Dataplane, error) {
	haCmd, err := runCommand(sd,
		cfg.HAProxyPath,
		"-f",
		cfg.HAProxyConfigPath,
	)
	if err != nil {
		return nil, err
	}

	_, err = runCommand(sd,
		cfg.DataplanePath,
		"--scheme", "unix",
		"--socket-path", cfg.DataplaneSock,
		"--haproxy-bin", cfg.HAProxyPath,
		"--config-file", cfg.HAProxyConfigPath,
		"--reload-cmd", fmt.Sprintf("kill -SIGUSR2 %d", haCmd.Process.Pid),
		"--reload-delay", "1",
		"--userlist", "controller",
		"--transaction-dir", cfg.DataplaneTransactionDir,
	)
	if err != nil {
		return nil, err
	}

	dataplaneClient := dataplane.New(
		"http://unix-sock",
		cfg.DataplaneUser,
		cfg.DataplanePass,
		&http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				Dial: func(proto, addr string) (conn net.Conn, err error) {
					return net.Dial("unix", cfg.DataplaneSock)
				},
			},
		},
	)

	// wait for startup
	for i := time.Duration(0); i < (5*time.Second)/(100*time.Millisecond); i++ {
		select {
		case <-sd.Stop:
			return nil, fmt.Errorf("exited")
		default:
		}

		err = dataplaneClient.Ping()
		if err != nil {
			fmt.Println("*****\n* SOUCHAY: wait for dataplane to be up\n*****")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		break
	}
	if err != nil {
		return nil, fmt.Errorf("timeout waiting for dataplaneapi: %s", err)
	}

	return dataplaneClient, nil
}

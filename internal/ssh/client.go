package ssh

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"

	l "github.com/SisyphusSQ/summary-sys/pkg/log"
)

type Client struct {
	config *Config
	conn   *ssh.Client
}

type Config struct {
	Host       string
	Port       int
	User       string
	AuthMethod AuthMethod
	Timeout    time.Duration
}

func NewClient(cfg *Config) (*Client, error) {
	if cfg.AuthMethod == nil {
		return nil, fmt.Errorf("auth method is required")
	}

	auth, err := cfg.AuthMethod.build()
	if err != nil {
		return nil, err
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         cfg.Timeout,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	l.Logger.Debugf("ssh connected to %s", addr)
	return &Client{config: cfg, conn: conn}, nil
}

func (c *Client) Run(ctx context.Context, cmd string) (string, string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	session.Stdout = stdoutBuf
	session.Stderr = stderrBuf

	if err := session.Run(cmd); err != nil {
		return stdoutBuf.String(), stderrBuf.String(), err
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}

func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Host() string {
	return c.config.Host
}

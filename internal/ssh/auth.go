package ssh

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type AuthMethod interface {
	build() (ssh.AuthMethod, error)
	Available() bool
}

func (a AgentAuth) Available() bool {
	return os.Getenv("SSH_AUTH_SOCK") != ""
}

func (p PasswordAuth) Available() bool {
	return p.Password != ""
}

func (k KeyAuth) Available() bool {
	return k.KeyPath != ""
}

func (d DefaultKeyAuth) Available() bool {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		user, err := user.Current()
		if err != nil {
			return false
		}
		homeDir = user.HomeDir
	}

	defaultKeys := []string{
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		filepath.Join(homeDir, ".ssh", "id_dsa"),
	}

	for _, keyPath := range defaultKeys {
		if _, err := os.Stat(keyPath); err == nil {
			return true
		}
	}
	return false
}

type PasswordAuth struct {
	Password string
}

func (p PasswordAuth) build() (ssh.AuthMethod, error) {
	return ssh.Password(p.Password), nil
}

type KeyAuth struct {
	KeyPath    string
	Passphrase string
}

func (k KeyAuth) build() (ssh.AuthMethod, error) {
	key, err := os.ReadFile(k.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	var signer ssh.Signer
	if k.Passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(k.Passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse key with passphrase: %w", err)
		}
	} else {
		signer, err = ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
	}

	return ssh.PublicKeys(signer), nil
}

type AgentAuth struct{}

func (a AgentAuth) build() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("dial SSH_AUTH_SOCK: %w", err)
	}

	client := agent.NewClient(conn)
	signers, err := client.Signers()
	if err != nil {
		return nil, fmt.Errorf("get SSH agent signers: %w", err)
	}

	return ssh.PublicKeys(signers...), nil
}

type DefaultKeyAuth struct{}

func (d DefaultKeyAuth) build() (ssh.AuthMethod, error) {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		user, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("get current user: %w", err)
		}
		homeDir = user.HomeDir
	}

	defaultKeys := []string{
		filepath.Join(homeDir, ".ssh", "id_rsa"),
		filepath.Join(homeDir, ".ssh", "id_ed25519"),
		filepath.Join(homeDir, ".ssh", "id_ecdsa"),
		filepath.Join(homeDir, ".ssh", "id_dsa"),
	}

	var signers []ssh.Signer
	for _, keyPath := range defaultKeys {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			continue
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			continue
		}
		signers = append(signers, signer)
	}

	if len(signers) == 0 {
		return nil, fmt.Errorf("no default SSH key found")
	}

	return ssh.PublicKeys(signers...), nil
}

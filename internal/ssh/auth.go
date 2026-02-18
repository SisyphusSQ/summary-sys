package ssh

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

type AuthMethod interface {
	build() (ssh.AuthMethod, error)
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
	return nil, fmt.Errorf("agent auth not implemented: use key or password auth")
}

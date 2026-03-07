package database

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/ayuzaka/sqls/dialect"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Proto string

const (
	ProtoTCP  Proto = "tcp"
	ProtoUDP  Proto = "udp"
	ProtoUnix Proto = "unix"
	ProtoHTTP Proto = "http"
)

type DBConfig struct {
	Alias          string                 `json:"alias" yaml:"alias"`
	Driver         dialect.DatabaseDriver `json:"driver" yaml:"driver"`
	DataSourceName string                 `json:"dataSourceName" yaml:"dataSourceName"`
	Proto          Proto                  `json:"proto" yaml:"proto"`
	User           string                 `json:"user" yaml:"user"`
	Passwd         string                 `json:"passwd" yaml:"passwd"`
	Host           string                 `json:"host" yaml:"host"`
	Port           int                    `json:"port" yaml:"port"`
	Path           string                 `json:"path" yaml:"path"`
	DBName         string                 `json:"dbName" yaml:"dbName"`
	Params         map[string]string      `json:"params" yaml:"params"`
	SSHCfg         *SSHConfig             `json:"sshConfig" yaml:"sshConfig"`
}

func (c *DBConfig) Validate() error {
	if c.Driver == "" {
		return errors.New("required: connections[].driver")
	}

	switch c.Driver {
	case
		dialect.DatabaseDriverMySQL,
		dialect.DatabaseDriverMySQL8,
		dialect.DatabaseDriverMySQL57,
		dialect.DatabaseDriverMySQL56,
		dialect.DatabaseDriverPostgreSQL,
		dialect.DatabaseDriverVertica:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}

		if c.DataSourceName == "" && c.Proto != "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			switch c.Proto {
			case ProtoTCP, ProtoUDP, ProtoHTTP:
				if c.Host == "" {
					return errors.New("required: connections[].host")
				}
			case ProtoUnix:
				if c.Path == "" {
					return errors.New("required: connections[].path")
				}
			default:
				return errors.New("invalid: connections[].proto")
			}
			if c.SSHCfg != nil {
				return c.SSHCfg.Validate()
			}
		}
	case dialect.DatabaseDriverSQLite3:
	case dialect.DatabaseDriverH2:
		if c.DataSourceName == "" {
			return errors.New("required: connections[].dataSourceName")
		}
	case dialect.DatabaseDriverMssql:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}
		if c.DataSourceName == "" && c.Proto != "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			switch c.Proto {
			case ProtoTCP:
				if c.Host == "" {
					return errors.New("required: connections[].host")
				}
			case ProtoUDP, ProtoUnix, ProtoHTTP:
			default:
				return errors.New("invalid: connections[].proto")
			}
		}
	case dialect.DatabaseDriverOracle:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}
		if c.DataSourceName == "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			if c.Passwd == "" {
				return errors.New("required: connections[].Passwd")
			}
			if c.Host == "" {
				return errors.New("required: connections[].Host")
			}
			if c.Port <= 0 {
				return errors.New("required: connections[].Port")
			}
			if c.DBName == "" {
				return errors.New("required: connections[].DBName")
			}
		}
	case dialect.DatabaseDriverClickhouse:
		if c.DataSourceName == "" && c.Proto == "" {
			return errors.New("required: connections[].dataSourceName or connections[].proto")
		}

		if c.DataSourceName == "" && c.Proto != "" {
			if c.User == "" {
				return errors.New("required: connections[].user")
			}
			switch c.Proto {
			case ProtoTCP, ProtoHTTP:
				if c.Host == "" {
					return errors.New("required: connections[].host")
				}
			case ProtoUDP, ProtoUnix:
			default:
				return errors.New("invalid: connections[].proto")
			}
			if c.SSHCfg != nil {
				return c.SSHCfg.Validate()
			}
		}

	default:
		return errors.New("invalid: connections[].driver")
	}
	return nil
}

type SSHConfig struct {
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	User       string `json:"user" yaml:"user"`
	PassPhrase string `json:"passPhrase" yaml:"passPhrase"`
	PrivateKey string `json:"privateKey" yaml:"privateKey"`
	UseAgent   bool   `json:"useAgent" yaml:"useAgent"`
	AgentSock  string `json:"agentSock" yaml:"agentSock"`
}

func (s *SSHConfig) Validate() error {
	if s.Host == "" {
		return errors.New("required: connections[]sshConfig.host")
	}
	if s.User == "" {
		return errors.New("required: connections[].sshConfig.user")
	}
	if s.PrivateKey == "" && !s.UseAgent {
		return errors.New("required: connections[].sshConfig.privateKey or connections[].sshConfig.useAgent")
	}
	return nil
}

func (s *SSHConfig) Endpoint() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func expandTildePath(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	if len(path) > 1 && path[1] != '/' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot expand home dir: %w", err)
	}
	if len(path) == 1 {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}

func (s *SSHConfig) ClientConfig() (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	if s.UseAgent {
		sockPath := s.AgentSock
		if sockPath == "" {
			sockPath = os.Getenv("SSH_AUTH_SOCK")
		}
		sockPath, err := expandTildePath(sockPath)
		if err != nil {
			return nil, err
		}
		if sockPath == "" {
			return nil, errors.New("SSH agent requested but SSH_AUTH_SOCK is not set and no agentSock configured")
		}
		conn, err := net.Dial("unix", sockPath)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to SSH agent at %s: %w", sockPath, err)
		}
		agentClient := agent.NewClient(conn)
		authMethods = append(authMethods, ssh.PublicKeysCallback(agentClient.Signers))
	}

	if s.PrivateKey != "" {
		buffer, err := os.ReadFile(s.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("cannot read SSH private key file, PrivateKey=%s, %w", s.PrivateKey, err)
		}
		var key ssh.Signer
		if s.PassPhrase != "" {
			key, err = ssh.ParsePrivateKeyWithPassphrase(buffer, []byte(s.PassPhrase))
			if err != nil {
				return nil, fmt.Errorf("cannot parse SSH private key file with passphrase, PrivateKey=%s, %w", s.PrivateKey, err)
			}
		} else {
			key, err = ssh.ParsePrivateKey(buffer)
			if err != nil {
				return nil, fmt.Errorf("cannot parse SSH private key file, PrivateKey=%s, %w", s.PrivateKey, err)
			}
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}

	sshConfig := &ssh.ClientConfig{
		User:            s.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return sshConfig, nil
}

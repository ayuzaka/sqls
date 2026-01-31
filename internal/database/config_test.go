package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSSHConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     SSHConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "useAgent only",
			cfg: SSHConfig{
				Host:     "bastion.example.com",
				Port:     22,
				User:     "sshuser",
				UseAgent: true,
			},
			wantErr: false,
		},
		{
			name: "useAgent with custom agentSock",
			cfg: SSHConfig{
				Host:      "bastion.example.com",
				Port:      22,
				User:      "sshuser",
				UseAgent:  true,
				AgentSock: "~/.1password/agent.sock",
			},
			wantErr: false,
		},
		{
			name: "useAgent with privateKey fallback",
			cfg: SSHConfig{
				Host:       "bastion.example.com",
				Port:       22,
				User:       "sshuser",
				UseAgent:   true,
				PrivateKey: "/home/user/.ssh/id_rsa",
			},
			wantErr: false,
		},
		{
			name: "no useAgent and no privateKey",
			cfg: SSHConfig{
				Host: "bastion.example.com",
				Port: 22,
				User: "sshuser",
			},
			wantErr: true,
			errMsg:  "required: connections[].sshConfig.privateKey or connections[].sshConfig.useAgent",
		},
		{
			name: "privateKey only (existing pattern)",
			cfg: SSHConfig{
				Host:       "bastion.example.com",
				Port:       22,
				User:       "sshuser",
				PrivateKey: "/home/user/.ssh/id_rsa",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			cfg: SSHConfig{
				Port:     22,
				User:     "sshuser",
				UseAgent: true,
			},
			wantErr: true,
			errMsg:  "required: connections[]sshConfig.host",
		},
		{
			name: "missing user",
			cfg: SSHConfig{
				Host:     "bastion.example.com",
				Port:     22,
				UseAgent: true,
			},
			wantErr: true,
			errMsg:  "required: connections[].sshConfig.user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("error message mismatch, want: %q, got: %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestExpandTildePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name: "tilde with slash expands to home directory",
			path: "~/.1password/agent.sock",
			want: filepath.Join(home, ".1password/agent.sock"),
		},
		{
			name: "tilde only expands to home directory",
			path: "~",
			want: home,
		},
		{
			name: "absolute path is unchanged",
			path: "/var/run/agent.sock",
			want: "/var/run/agent.sock",
		},
		{
			name: "empty path is unchanged",
			path: "",
			want: "",
		},
		{
			name: "~user path is returned as-is",
			path: "~otheruser/.ssh/id_rsa",
			want: "~otheruser/.ssh/id_rsa",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandTildePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expandTildePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("expandTildePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

package database

import (
	"testing"
)

func Test_replaceNetForSSH(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		want    string
		wantErr bool
	}{
		{
			name: "tcp network is replaced with mysql+tcp",
			dsn:  "user:password@tcp(192.168.1.1:3306)/dbname",
			want: "user:password@mysql+tcp(192.168.1.1:3306)/dbname",
		},
		{
			name: "dataSourceName format",
			dsn:  "admin:secret@tcp(db.example.com:3306)/test",
			want: "admin:secret@mysql+tcp(db.example.com:3306)/test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceNetForSSH(tt.dsn)
			if (err != nil) != tt.wantErr {
				t.Errorf("replaceNetForSSH() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("replaceNetForSSH() = %q, want %q", got, tt.want)
			}
		})
	}
}

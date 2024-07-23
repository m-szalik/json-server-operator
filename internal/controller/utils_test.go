package controller

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_validateJson(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"valid json", `{"k":"value","x":172}`, false},
		{"invalid json", `value","x":172}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateJson(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("validateJson() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_md5sum(t *testing.T) {
	tests := []struct {
		args string
		hash string
	}{
		{"hey joe", `3d627867e420b42d7c82f56c474c0ede`},
		{"", `d41d8cd98f00b204e9800998ecf8427e`},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("md5sum-%s", tt.hash), func(t *testing.T) {
			hash := md5hash(tt.args)
			assert.Equal(t, tt.hash, hash)
		})
	}
}

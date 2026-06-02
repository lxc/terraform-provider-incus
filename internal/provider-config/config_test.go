package config_test

import (
	"slices"
	"testing"

	config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

func TestRemoteAddresses(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    []string
	}{
		{
			name:    "empty",
			address: "",
			want:    nil,
		},
		{
			name:    "single https address",
			address: "https://10.29.227.20:8443",
			want:    []string{"https://10.29.227.20:8443"},
		},
		{
			name:    "multiple https addresses",
			address: "https://10.29.227.20:8443,https://127.0.0.1:8443",
			want:    []string{"https://10.29.227.20:8443", "https://127.0.0.1:8443"},
		},
		{
			name:    "trims whitespace and ignores empty entries",
			address: " https://10.29.227.20:8443, , https://127.0.0.1:8443 ",
			want:    []string{"https://10.29.227.20:8443", "https://127.0.0.1:8443"},
		},
		{
			name:    "unix socket",
			address: "unix://",
			want:    []string{"unix://"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := config.RemoteAddresses(tt.address)
			if tt.want == nil && got != nil {
				t.Fatalf("RemoteAddresses() = %#v, want nil", got)
			}

			if tt.want != nil && !slices.Equal(got, tt.want) {
				t.Fatalf("RemoteAddresses() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

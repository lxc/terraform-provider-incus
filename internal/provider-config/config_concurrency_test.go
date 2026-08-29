package config

import (
	"sync"
	"testing"

	incus_config "github.com/lxc/incus/v7/shared/cliconfig"
)

func TestGetIncusConfigInstanceServerConcurrent(t *testing.T) {
	const attempts = 10
	const clients = 64
	const remoteName = "test"

	for range attempts {
		incusConfig := incus_config.NewConfig(t.TempDir(), false)
		incusConfig.Remotes = map[string]incus_config.Remote{
			remoteName: {
				Addrs:    []string{"https://127.0.0.1:0"},
				Protocol: "incus",
			},
		}

		provider := NewIncusProvider(incusConfig, false)
		start := make(chan struct{})
		var wg sync.WaitGroup

		for range clients {
			wg.Go(func() {
				<-start
				_, _ = provider.getIncusConfigInstanceServer(remoteName)
			})
		}

		close(start)
		wg.Wait()

		if incusConfig.Remotes[remoteName].TLS == nil {
			t.Fatal("client certificate was not cached")
		}
	}
}

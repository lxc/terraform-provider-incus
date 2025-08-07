package acctest

import (
	"testing"

	"github.com/lxc/incus/v6/shared/api"
)

// PreConfigAccTestServerConfig ensures the presence of the server config key:
//
//	logging.acctest-pre-existing.target.type: "loki"
//
// This is required for the tests of the server resource, since it is required
// to preserve the pre existing config keys.
func PreConfigAccTestServerConfig(t *testing.T, shouldRemain bool) {
	t.Helper()

	p := testProvider()
	client, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	server, eTag, err := client.GetServer()
	if err != nil {
		t.Fatal(err)
	}

	server.Config["logging.acctest-pre-existing.target.type"] = "loki"

	err = client.UpdateServer(api.ServerPut{
		Config: server.Config,
	}, eTag)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		server, eTag, err := client.GetServer()
		if err != nil {
			t.Fatal(err)
		}

		val, ok := server.Config["logging.acctest-pre-existing.target.type"]

		if shouldRemain {
			if !ok || val != "loki" {
				t.Errorf("logging.acctest-pre-existing.target.type is changed or gone, where it was expected to stay (value: %s)", val)
			}
		} else {
			if ok {
				t.Error("logging.acctest-pre-existing.target.type is still present where it was was expected for it to be gone")
			}
		}

		// Ensure in either case, that we properly cleanup.
		delete(server.Config, "logging.acctest-pre-existing.target.type")

		err = client.UpdateServer(api.ServerPut{
			Config: server.Config,
		}, eTag)
		if err != nil {
			t.Fatal(err)
		}
	})
}

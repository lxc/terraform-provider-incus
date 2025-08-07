package acctest

import (
	"testing"

	"github.com/lxc/incus/v6/shared/api"
)

// PreConfigAccTestServerConfig ensures the presence of the server config key:
//
//	user.acctest-pre-existing.key: "value"
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

	server.Config["user.acctest-pre-existing.key"] = "value"

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

		val, ok := server.Config["user.acctest-pre-existing.key"]

		if shouldRemain {
			if !ok || val != "value" {
				t.Errorf(`user.acctest-pre-existing.key is changed or gone, where it was expected to stay (expected value: "value", got: %q)`, val)
			}
		} else {
			if ok {
				t.Errorf("user.acctest-pre-existing.key is still present where it was was expected for it to be gone (got: %q)", val)
			}
		}

		// Ensure in either case, that we properly cleanup.
		server.Config["user.acctest-pre-existing.key"] = ""

		err = client.UpdateServer(api.ServerPut{
			Config: server.Config,
		}, eTag)
		if err != nil {
			t.Fatal(err)
		}
	})
}

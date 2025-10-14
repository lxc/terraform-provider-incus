package acctest

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/lxc/terraform-provider-incus/internal/utils"
)

// PreCheck is a precheck that ensures test requirements, such as existing
// environment variables, are met. It should be included in every acc test.
func PreCheck(t *testing.T) {
	t.Helper()
	// if os.Getenv("TEST_INCUS_REQUIRED_VAR") == "" {
	// 	t.Fatal("TEST_INCUS_REQUIRED_VAR must be set for acceptance tests")
	// }
}

// PreCheckIncusVersion skips the test if the server's version does not satisfy
// the provided version constraints. The version constraints are detailed at:
// https://pkg.go.dev/github.com/hashicorp/go-version#readme-version-constraints
func PreCheckIncusVersion(t *testing.T, versionConstraint string) {
	t.Helper()

	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		t.Fatal(err)
	}

	serverVersion := apiServer.Environment.ServerVersion
	ok, err := utils.CheckVersion(serverVersion, versionConstraint)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Skipf("Test %q skipped. Incus server version %q does not satisfy the version constraint %q", t.Name(), serverVersion, versionConstraint)
	}
}

// PreCheckAPIExtensions skips the test if the Incus server does not support
// the required extensions.
func PreCheckAPIExtensions(t *testing.T, extensions ...string) {
	t.Helper()

	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	missing := []string{}
	for _, e := range extensions {
		if !server.HasExtension(e) {
			missing = append(missing, e)
		}
	}

	if len(missing) > 0 {
		t.Skipf("Test %q skipped. Incus server is missing required extensions: %v", t.Name(), missing)
	}
}

// PreCheckVirtualization skips the test if the Incus server does not
// support virtualization.
func PreCheckVirtualization(t *testing.T) {
	t.Helper()

	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skipf("Test %q skipped. Virtualization tests can't run in Github Actions.", t.Name())
	}

	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		t.Fatal(err)
	}

	// Ensure that Incus server supports qemu driver which is required for virtualization.
	if !strings.Contains(apiServer.Environment.Driver, "qemu") {
		t.Skipf("Test %q skipped. Incus server does not support virtualization.", t.Name())
	}
}

// PreCheckClustering skips the test if Incus server is not running
// in clustered mode.
func PreCheckClustering(t *testing.T) {
	t.Helper()

	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if !server.IsClustered() {
		t.Skipf("Test %q skipped. Incus server is not running in clustered mode.", t.Name())
	}
}

// PreCheckStandalone skips the test if Incus server is running
// in clustered mode.
func PreCheckStandalone(t *testing.T) {
	t.Helper()

	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if server.IsClustered() {
		t.Skipf("Test %q skipped. Incus server is running in clustered mode.", t.Name())
	}
}

func PreCheckX86_64(t *testing.T) {
	t.Helper()

	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		t.Fatal(err)
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		t.Fatal(err)
	}

	for _, arch := range apiServer.Environment.Architectures {
		if arch == "x86_64" {
			return
		}
	}

	t.Skipf("Test %q skipped: Incus server does not support x86_64.", t.Name())
}

// PrintResourceState is a test check function that prints the entire state
// of a resource with the given name. This check should be used only for
// debuging purposes.
//
// Example resource name:
//
//	incus_profile.profile2
func PrintResourceState(t *testing.T, resName string) resource.TestCheckFunc {
	t.Helper()

	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resName]
		if !ok {
			return fmt.Errorf("Resource %q not found", resName)
		}

		fmt.Println(utils.ToPrettyJSON(rs))
		return nil
	}
}

// TestCheckGetClusterMemberNames populates the provided targetClusterMemberNames
// map with the names of the cluster members detected by the incus_cluster data
// source addressed by "name".
func TestCheckGetClusterMemberNames(t *testing.T, name string, targetClusterMembers map[string]struct{}) resource.TestCheckFunc {
	t.Helper()

	return func(s *terraform.State) error {
		ms := s.RootModule()

		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s in %s", name, ms.Path)
		}

		if rs.Type != "incus_cluster" {
			return fmt.Errorf("TestCheckGetClusterMembers can only be used with data souce incus_cluster")
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s in %s", name, ms.Path)
		}

		for k := range is.Attributes {
			if !strings.HasPrefix(k, "members.") {
				continue
			}

			parts := strings.Split(k, ".")

			if parts[1] == "%" {
				continue
			}

			targetClusterMembers[parts[1]] = struct{}{}
		}

		return nil
	}
}

// TestCheckResourceAttrInLookup ensures a value stored in state for the given
// name and key combination, is checked against a lookup map.
// This check is successful, if in the lookup map a key for the state value
// exists.
func TestCheckResourceAttrInLookup(name string, key string, lookup map[string]struct{}) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(name, key, func(value string) error {
		_, ok := lookup[value]
		if !ok {
			return fmt.Errorf("value %q not found in lookup (%v)", value, lookup)
		}

		return nil
	})
}

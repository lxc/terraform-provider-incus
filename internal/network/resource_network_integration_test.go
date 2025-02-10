package network_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkIntegration_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_integrations")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkIntegration_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_integration.test", "name", "test"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "description", "Basic Network Integration"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccNetworkIntegration_withConfig(t *testing.T) {
	networkIntegrationConfig := map[string]string{
		"ovn.northbound_connection": "tcp:[192.0.2.12]:6645,tcp:[192.0.3.13]:6645,tcp:[192.0.3.14]:6645",
		"ovn.southbound_connection": "tcp:[192.0.2.12]:6646,tcp:[192.0.3.13]:6646,tcp:[192.0.3.14]:6646",
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_integrations")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkIntegration_withConfig(networkIntegrationConfig),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_integration.test", "name", "test"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "config.%", "2"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "config.ovn.northbound_connection", networkIntegrationConfig["ovn.northbound_connection"]),
					resource.TestCheckResourceAttr("incus_network_integration.test", "config.ovn.southbound_connection", networkIntegrationConfig["ovn.southbound_connection"]),
				),
			},
		},
	})
}

func TestAccNetworkIntegration_withValidType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_integrations")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkIntegration_withType("ovn"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_integration.test", "name", "test"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network_integration.test", "config.%", "0"),
				),
			},
		},
	})
}

func TestAccNetworkIntegration_withInvalidType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_integrations")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccNetworkIntegration_withType("invalid"),
				ExpectError: regexp.MustCompile(`Attribute type value must be one of: \["ovn"\], got: "invalid"`),
			},
		},
	})
}

func TestAccNetworkIntegration_attach(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_integrations")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkIntegration_attach(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_integration.test", "name", "test"),
				),
			},
		},
	})
}

func testAccNetworkIntegration_basic() string {
	return `
resource "incus_network_integration" "test" {
  name        = "test"
  description = "Basic Network Integration"
  type        = "ovn"
}
`
}

func testAccNetworkIntegration_withConfig(config map[string]string) string {
	entries := strings.Builder{}
	for k, v := range config {
		entry := fmt.Sprintf("%q = %q\n", k, v)
		entries.WriteString(entry)
	}

	return fmt.Sprintf(`
resource "incus_network_integration" "test" {
  name        = "test"
  description = "Network Integration with Config"
  type        = "ovn"

  config = {
    %s
  }
}
`, entries.String())
}

func testAccNetworkIntegration_withType(networkIntegrationType string) string {
	return fmt.Sprintf(`
resource "incus_network_integration" "test" {
  name        = "test"
  description = "Network Integration with Type"
  type        = "%s"
}
`, networkIntegrationType)
}

func testAccNetworkIntegration_attach() string {
	networkIntegrationConfig := map[string]string{
		"ovn.northbound_connection": "unix:/var/run/ovn/ovn_ic_nb_db.sock",
		"ovn.southbound_connection": "unix:/var/run/ovn/ovn_ic_sb_db.sock",
	}

	networkIntegrationRes := `
resource "incus_network_peer" "test" {
  name               = "ovn-lan1"
  network            = incus_network.ovn.name
  target_integration = incus_network_integration.test.name
  type               = "remote"
}
`
	return fmt.Sprintf("%s\n%s\n%s", ovnNetworkResource(), testAccNetworkIntegration_withConfig(networkIntegrationConfig), networkIntegrationRes)
}

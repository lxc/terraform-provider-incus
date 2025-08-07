package network_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkPeer_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_peer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkPeer_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("incus_network.ovnbr", "type", "bridge"),
					resource.TestCheckResourceAttr("incus_network.lan0", "name", "lan0"),
					resource.TestCheckResourceAttr("incus_network.lan0", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network.lan0", "config.ipv4.address", "10.0.0.1/24"),
					resource.TestCheckResourceAttr("incus_network.lan1", "name", "lan1"),
					resource.TestCheckResourceAttr("incus_network.lan1", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network.lan1", "config.ipv4.address", "10.0.1.1/24"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "name", "lab0-lan1"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "network", "lan0"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "target_network", "lan1"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "config.%", "0"),
				),
			},
		},
	})
}

// Creates a network peering.
func testAccNetworkPeer_basic() string {
	return fmt.Sprintf(`
%s%s%s

resource "incus_network_peer" "lan0_lan1"{
	name = "lab0-lan1"
	network = incus_network.lan0.name
	target_network = incus_network.lan1.name
}

resource "incus_network_peer" "lan1_lan0"{
	name = "lab1-lan0"
	network = incus_network.lan1.name
	target_network = incus_network.lan0.name
}
	`,
		testAccNetworkPeer_ovnbr(),
		testAccNetworkPeer_network("lan0", "10.0.0.1"),
		testAccNetworkPeer_network("lan1", "10.0.1.1"))
}

func TestAccNetworkPeer_acrossProjects(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "network_peer")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkPeer_acrossProjects(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network.ovnbr", "name", "ovnbr"),
					resource.TestCheckResourceAttr("incus_network.ovnbr", "type", "bridge"),
					resource.TestCheckResourceAttr("incus_project.projectA", "name", "projectA"),
					resource.TestCheckResourceAttr("incus_project.projectB", "name", "projectB"),
					resource.TestCheckResourceAttr("incus_network.projectA_lan0", "name", "lan0"),
					resource.TestCheckResourceAttr("incus_network.projectA_lan0", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network.projectA_lan0", "config.ipv4.address", "10.0.0.1/24"),
					resource.TestCheckResourceAttr("incus_network.projectB_lan1", "name", "lan1"),
					resource.TestCheckResourceAttr("incus_network.projectB_lan1", "type", "ovn"),
					resource.TestCheckResourceAttr("incus_network.projectB_lan1", "config.ipv4.address", "10.0.1.1/24"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "name", "lab0-lan1"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "network", "lan0"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "target_network", "lan1"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "target_project", "projectB"),
					resource.TestCheckResourceAttr("incus_network_peer.lan0_lan1", "config.%", "0"),
				),
			},
		},
	})
}

// Creates a network peering between projects.
func testAccNetworkPeer_acrossProjects() string {
	return fmt.Sprintf(`
%s%s%s%s%s

resource "incus_network_peer" "lan0_lan1"{
	name = "lab0-lan1"
	project = incus_project.projectA.name
	network = incus_network.projectA_lan0.name
	target_project = incus_project.projectB.name
	target_network = incus_network.projectB_lan1.name
}

resource "incus_network_peer" "lan1_lan0"{
	name = "lab1-lan0"
	project = incus_project.projectB.name
	network = incus_network.projectB_lan1.name
	target_project = incus_project.projectA.name
	target_network = incus_network.projectA_lan0.name
}
	`,
		testAccNetworkPeer_ovnbr(),
		testAccNetworkPeer_project("projectA"),
		testAccNetworkPeer_project("projectB"),
		testAccNetworkPeer_projectNetwork("projectA", "lan0", "10.0.0.1"),
		testAccNetworkPeer_projectNetwork("projectB", "lan1", "10.0.1.1"))
}

func testAccNetworkPeer_ovnbr() string {
	return `
resource "incus_network" "ovnbr" {
  name = "ovnbr"
  type = "bridge"
  config = {
    "ipv4.address"     = "10.10.10.1/24"
    "ipv4.routes"      = "10.10.10.192/26"
    "ipv4.ovn.ranges"  = "10.10.10.193-10.10.10.254"
    "ipv4.dhcp.ranges" = "10.10.10.100-10.10.10.150"
    "ipv6.address"     = "fd42:1000:1000:1000::1/64"
    "ipv6.dhcp.ranges" = "fd42:1000:1000:1000:a::-fd42:1000:1000:1000:a::ffff"
    "ipv6.ovn.ranges"  = "fd42:1000:1000:1000:b::-fd42:1000:1000:1000:b::ffff"
  }
}
`
}

func testAccNetworkPeer_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "%s" {
	name = "%s"
	config = {
    "features.networks"       = true
  }
}
`, project, project)
}

func testAccNetworkPeer_network(network string, ipv4 string) string {
	return fmt.Sprintf(`
resource "incus_network" "%s" {
  name = "%s"
  type = "ovn"

  config = {
    "ipv4.address" = "%s/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "none"
    "ipv6.nat"     = "false"
    "network" = incus_network.ovnbr.name
  }
}
`, network, network, ipv4)
}

func testAccNetworkPeer_projectNetwork(project string, network string, ipv4 string) string {
	return fmt.Sprintf(`
resource "incus_network" "%s_%s" {
  name = "%s"
  type = "ovn"
  project = incus_project.%s.name

  config = {
    "ipv4.address" = "%s/24"
    "ipv4.nat"     = "true"
    "ipv6.address" = "none"
    "ipv6.nat"     = "false"
    "network" = incus_network.ovnbr.name
  }
}
`, project, network, network, project, ipv4)
}

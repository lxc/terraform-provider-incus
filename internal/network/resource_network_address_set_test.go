package network_test

import (
	"fmt"
	"regexp"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccNetworkAddressSet_basic(t *testing.T) {
	name := petname.Generate(2, "-")
	description := "Network address set"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkAddressSet(name, description),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_network_address_set.this", "name", name),
					resource.TestCheckResourceAttr("incus_network_address_set.this", "description", description),
					resource.TestCheckResourceAttr("incus_network_address_set.this", "addresses.#", "2"),
				),
			},
		},
	})
}

func TestAccNetworkAddressSet_withoutAddresses(t *testing.T) {
	name := petname.Generate(2, "-")
	description := "Network address set"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccNetworkAddressSetWithoutAddresses(name, description),
				ExpectError: regexp.MustCompile(`The argument "addresses" is required, but no definition was found.`),
			},
		},
	})
}

func testAccNetworkAddressSet(name, description string) string {
	return fmt.Sprintf(`
resource "incus_network_address_set" "this" {
	name        = "%s"
	description = "%s"
	addresses   = ["10.0.0.2", "10.0.0.3"]
}
`, name, description)
}

func testAccNetworkAddressSetWithoutAddresses(name, description string) string {
	return fmt.Sprintf(`
resource "incus_network_address_set" "this" {
	name        = "%s"
	description = "%s"
}
`, name, description)
}

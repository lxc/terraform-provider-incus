package server_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccServer_create_update_delete(t *testing.T) {
	configUserName := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccServer1(configLoggingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName), "password"),
				),
			},
			{
				Config: testAccServer2(configUserName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user_new"),
					resource.TestCheckNoResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName)),
				),
			},
		},
	})
}

func TestAccServer_empty(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "incus_server" "test" {
}
`,
			},
		},
	})
}

func testAccServer1(configLoggingName string) string {
	return fmt.Sprintf(`
resource "incus_server" "%[1]s" {
  config = {
    "user.%[1]s.create" = "value"
    "user.%[1]s.update" = "value"
    "user.%[1]s.remove" = "value"
  }
}
`, configLoggingName)
}

func testAccServer2(configLoggingName string) string {
	return fmt.Sprintf(`
resource "incus_server" "%[1]s" {
  config = {
    "user.%[1]s.create"    = "value"
    "user.%[1]s.update"    = "new_value" // updated
    // "user.%[1]s.remove" = "value" // removed (commented out)
  }
}
`, configLoggingName)
}

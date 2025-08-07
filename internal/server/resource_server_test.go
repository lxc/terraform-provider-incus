package server_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccServer_create_update_delete(t *testing.T) {
	configLoggingName := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, true) }, // ensures, that "logging.acctest-pre-existing.target.type" already exists in the config and that it remains of destroy.
				Config:    testAccServer1(configLoggingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName), "password"),
					resource.TestCheckNoResourceAttr("incus_server.test", "config.logging.acctest-pre-existing.target.type"), // pre existing, not managed through Terraform.
				),
			},
			{
				Config: testAccServer2(configLoggingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user_new"),
					resource.TestCheckNoResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName)),
					resource.TestCheckNoResourceAttr("incus_server.test", "config.logging.acctest-pre-existing.target.type"), // pre existing, not managed through Terraform.
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

func TestAccServer_create_overwrite_pre_existing(t *testing.T) {
	configLoggingName := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, false) }, // ensures, that "logging.acctest-pre-existing.target.type" already exists in the config. It is expected to be gone after the test.
				Config:    testAccServerPreExisting(configLoggingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName), "password"),
					resource.TestCheckResourceAttr("incus_server.test", "config.logging.acctest-pre-existing.target.type", "loki"), // pre existing, now managed through Terraform.
				),
			},
		},
	})
}

func TestAccServer_update_overwrite_pre_existing(t *testing.T) {
	configLoggingName := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, false) }, // ensures, that "logging.acctest-pre-existing.target.type" already exists in the config. It is expected to be gone after the test.
				Config:    testAccServer1(configLoggingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName), "password"),
					resource.TestCheckNoResourceAttr("incus_server.test", "config.logging.acctest-pre-existing.target.type"), // pre existing, not managed through Terraform.
				),
			},
			{
				Config: testAccServerPreExisting(configLoggingName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.type", configLoggingName), "loki"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.username", configLoggingName), "user"),
					resource.TestCheckResourceAttr("incus_server.test", fmt.Sprintf("config.logging.%s.target.password", configLoggingName), "password"),
					resource.TestCheckResourceAttr("incus_server.test", "config.logging.acctest-pre-existing.target.type", "loki"), // pre existing, now managed through Terraform.
				),
			},
		},
	})
}

func testAccServer1(configLoggingName string) string {
	return fmt.Sprintf(`
resource "incus_server" "test" {
  config = {
    "logging.%[1]s.target.type"     = "loki"
    "logging.%[1]s.target.username" = "user"
    "logging.%[1]s.target.password" = "password"
  }
}
`, configLoggingName)
}

func testAccServer2(configLoggingName string) string {
	return fmt.Sprintf(`
resource "incus_server" "test" {
  config = {
    "logging.%[1]s.target.type"        = "loki"
    "logging.%[1]s.target.username"    = "user_new" // updated
    // "logging.%[1]s.target.password" = "password" // removed (commented out)
  }
}
`, configLoggingName)
}

func testAccServerPreExisting(configLoggingName string) string {
	return fmt.Sprintf(`
resource "incus_server" "test" {
  config = {
    "logging.%[1]s.target.type"                = "loki"
    "logging.%[1]s.target.username"            = "user"
    "logging.%[1]s.target.password"            = "password"
    "logging.acctest-pre-existing.target.type" = "loki" // pre existing key, now managed through Terraform
  }
}
`, configLoggingName)
}

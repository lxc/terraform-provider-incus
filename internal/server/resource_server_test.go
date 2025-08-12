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
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, true) }, // ensures, that "acctest-pre-existing.key" already exists in the config and that it remains after destroy.
				Config:    testAccServer1(configUserName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.create", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.update", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.remove", configUserName), "value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), "config.user.acctest-pre-existing.key"), // pre existing, not managed through Terraform.
				),
			},
			{
				Config: testAccServer2(configUserName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.create", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.update", configUserName), "new_value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.remove", configUserName)),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), "config.user.acctest-pre-existing.key"), // pre existing, not managed through Terraform.
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
	configUserName := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, false) }, // ensures, that "user.acctest-pre-existing.key" already exists in the config. It is expected to be gone after the test.
				Config:    testAccServerPreExisting(configUserName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.create", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.update", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.remove", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), "config.user.acctest-pre-existing.key", "new_value"), // pre existing, now managed through Terraform.
				),
			},
		},
	})
}

func TestAccServer_update_overwrite_pre_existing(t *testing.T) {
	configUserName := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, false) }, // ensures, that "user.acctest-pre-existing.key" already exists in the config. It is expected to be gone after the test.
				Config:    testAccServer1(configUserName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.create", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.update", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.remove", configUserName), "value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), "config.user.acctest-pre-existing.key"), // pre existing, not managed through Terraform.
				),
			},
			{
				Config: testAccServerPreExisting(configUserName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.create", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.update", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), fmt.Sprintf("config.user.%s.remove", configUserName), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName), "config.user.acctest-pre-existing.key", "new_value"), // pre existing, now managed through Terraform.
				),
			},
		},
	})
}

func TestAccServer_multiple_resources(t *testing.T) {
	configUserName1 := "acctest-" + petname.Generate(2, "-")
	configUserName2 := "acctest-" + petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				PreConfig: func() { acctest.PreConfigAccTestServerConfig(t, true) }, // ensures, that "acctest-pre-existing.key" already exists in the config and that it remains after destroy.
				Config:    testAccMultipleResources1(configUserName1, configUserName2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.create", configUserName1), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.update", configUserName1), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.remove", configUserName1), "value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.create", configUserName2)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.update", configUserName2)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.remove", configUserName2)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), "config.user.acctest-pre-existing.key"),                // pre existing, not managed through Terraform.

					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.create", configUserName2), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.update", configUserName2), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.remove", configUserName2), "value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.create", configUserName1)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.update", configUserName1)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.remove", configUserName1)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), "config.user.acctest-pre-existing.key"),                // pre existing, not managed through Terraform.
				),
			},
			{
				Config: testAccMultipleResources2(configUserName1, configUserName2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.create", configUserName1), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.update", configUserName1), "new_value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.remove", configUserName1)),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.create", configUserName2)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.update", configUserName2)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), fmt.Sprintf("config.user.%s.remove", configUserName2)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName1), "config.user.acctest-pre-existing.key"),                // pre existing, not managed through Terraform.

					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.create", configUserName2), "value"),
					resource.TestCheckResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.update", configUserName2), "new_value"),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.remove", configUserName2)),
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.create", configUserName1)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.update", configUserName1)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), fmt.Sprintf("config.user.%s.remove", configUserName1)), // Not managed by this resource.
					resource.TestCheckNoResourceAttr(fmt.Sprintf("incus_server.%s", configUserName2), "config.user.acctest-pre-existing.key"),                // pre existing, not managed through Terraform.
				),
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

func testAccServerPreExisting(configLoggingName string) string {
	return fmt.Sprintf(`
resource "incus_server" "%[1]s" {
  config = {
    "user.%[1]s.create"             = "value"
		"user.%[1]s.update"             = "value"
    "user.%[1]s.remove"             = "value"
    "user.acctest-pre-existing.key" = "new_value" // pre existing key, now managed through Terraform
  }
}
`, configLoggingName)
}

func testAccMultipleResources1(configLoggingName1 string, configLoggingName2 string) string {
	return fmt.Sprintf(`
resource "incus_server" "%[1]s" {
  config = {
    "user.%[1]s.create" = "value"
    "user.%[1]s.update" = "value"
    "user.%[1]s.remove" = "value"
  }
}

resource "incus_server" "%[2]s" {
  config = {
    "user.%[2]s.create" = "value"
    "user.%[2]s.update" = "value"
    "user.%[2]s.remove" = "value"
  }
}
`, configLoggingName1, configLoggingName2)
}

func testAccMultipleResources2(configLoggingName1 string, configLoggingName2 string) string {
	return fmt.Sprintf(`
resource "incus_server" "%[1]s" {
  config = {
    "user.%[1]s.create"    = "value"
    "user.%[1]s.update"    = "new_value" // updated
    // "user.%[1]s.remove" = "value" // removed (commented out)
  }
}

resource "incus_server" "%[2]s" {
  config = {
    "user.%[2]s.create"    = "value"
    "user.%[2]s.update"    = "new_value" // updated
    // "user.%[2]s.remove" = "value" // removed (commented out)
  }
}
`, configLoggingName1, configLoggingName2)
}

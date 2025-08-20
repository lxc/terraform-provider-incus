package project_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

// At a high level, the first basic test for a resource should establish the following:
// - Terraform can plan and apply a common resource configuration without error.
// - Verify the expected attributes are saved to state, and contain the values expected.
// - Verify the values in the remote API/Service for the resource match what is stored in state.
// - Verify that a subsequent terraform plan does not produce a diff/change.

func TestAccProject_basic(t *testing.T) {
	projectName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_basic(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project0", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project0", "description", "Terraform provider test project"),
					resource.TestCheckResourceAttr("incus_project.project0", "force_destroy", "false"),
					// Ensure state of computed keys is not tracked.
					resource.TestCheckNoResourceAttr("incus_project.project0", "config.features.images"),
					resource.TestCheckNoResourceAttr("incus_project.project0", "config.features.profiles"),
					resource.TestCheckNoResourceAttr("incus_project.project0", "config.features.storage.volumes"),
					resource.TestCheckNoResourceAttr("incus_project.project0", "config.features.storage.buckets"),
				),
			},
		},
	})
}

func TestAccProject_config(t *testing.T) {
	projectName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_config(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.images", "true"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.profiles", "false"),
					resource.TestCheckResourceAttr("incus_project.project1", "force_destroy", "false"),
					// Ensure state of computed keys is not tracked.
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.volumes"),
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.buckets"),
				),
			},
		},
	})
}

func TestAccProject_updateConfig(t *testing.T) {
	projectName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_updateConfig_1(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "description", "Old description"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.images", "true"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.profiles", "false"),
					resource.TestCheckResourceAttr("incus_project.project1", "force_destroy", "false"),
					// Ensure state of computed keys is not tracked.
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.volumes"),
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.buckets"),
				),
			},
			{
				Config: testAccProject_updateConfig_2(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "description", "New description"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.images", "false"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.profiles", "true"),
					resource.TestCheckResourceAttr("incus_project.project1", "force_destroy", "false"),
					// Ensure state of computed keys is not tracked.
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.volumes"),
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.buckets"),
				),
			},
			{
				Config: testAccProject_updateConfig_3(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "description", "New description"),
					resource.TestCheckResourceAttr("incus_project.project1", "force_destroy", "true"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.images", "false"),
					resource.TestCheckResourceAttr("incus_project.project1", "config.features.profiles", "true"),
					// Ensure state of computed keys is not tracked.
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.volumes"),
					resource.TestCheckNoResourceAttr("incus_project.project1", "config.features.storage.buckets"),
				),
			},
		},
	})
}


// force_destroy is tested by creating a project containing an untracked instance
// and letting CheckDestroy verify that the project can be destroyed.
func TestAccProject_forceDestroy(t *testing.T) {
	projectName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_forceDestroy_config(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "force_destroy", "true"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "project", projectName),
				),
			},
			{
				Config: testAccProject_forceDestroy_danglingInstance(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_project.project1", "force_destroy", "true"),
				),
			},
		},
	})
}

func TestAccProject_importBasic(t *testing.T) {
	resourceName := "incus_project.project0"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_basic("project0"),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "project0",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func TestAccProject_importConfig(t *testing.T) {
	resourceName := "incus_project.project1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProject_config("project1"),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        "project1",
				ImportState:                          true,
				ImportStateVerify:                    false,
				ImportStateVerifyIdentifierAttribute: "name",
			},
		},
	})
}

func testAccProject_basic(name string) string {
	return fmt.Sprintf(`
resource "incus_project" "project0" {
  name        = "%s"
  description = "Terraform provider test project"
}`, name)
}

func testAccProject_config(name string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "Terraform provider test project"
  config = {
	"features.images"   = true
	"features.profiles" = false
  }
}`, name)
}

func testAccProject_updateConfig_1(name string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "Old description"
  config = {
	"features.images"   = true
	"features.profiles" = false
  }
}`, name)
}

func testAccProject_updateConfig_2(name string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "New description"
  config = {
	"features.images"   = false
	"features.profiles" = true
  }
}`, name)
}

func testAccProject_updateConfig_3(name string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%s"
  description = "New description"
  force_destroy = true
  config = {
	"features.images"   = false
	"features.profiles" = true
  }
}`, name)
}

func testAccProject_forceDestroy_config(name string, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name          = "%s"
  description   = "Terraform provider test project"
  force_destroy = true
  config = {
	"features.images"   = true
	"features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  name   = "%s"
  project = incus_project.project1.name
  type = "container"
  image  = "%s"
}
`, name, instanceName, acctest.TestImage)
}

func testAccProject_forceDestroy_danglingInstance(name string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name          = "%s"
  description   = "Terraform provider test project"
  force_destroy = true
  config = {
	"features.images"   = true
	"features.profiles" = false
  }
}

removed {
  from = incus_instance.instance1
  lifecycle {
	destroy = false
  }
}
`, name)
}



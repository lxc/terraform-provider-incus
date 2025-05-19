package image_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccImageAlias_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImageAlias_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image_alias.alias1", "alias", "alpine-test"),
					resource.TestCheckResourceAttr("incus_image_alias.alias1", "description", "Alpine Linux"),
				),
			},
		},
	})
}

func TestAccImageAlias_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImageAlias_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.copy_aliases", "false"),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("incus_image_alias.alias1", "alias", "alpine-test"),
					resource.TestCheckResourceAttr("incus_image_alias.alias1", "description", "Alpine Linux"),
					resource.TestCheckResourceAttr("incus_image_alias.alias1", "project", projectName),
				),
			},
		},
	})
}

func testAccImageAlias_basic() string {
	return `
resource "incus_image" "img1" {
  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = false
  }
}

resource "incus_image_alias" "alias1" {
  alias          = "alpine-test"
  description    = "Alpine Linux"
  fingerprint    = incus_image.img1.fingerprint

  depends_on = [incus_image.img1]
}
	`
}

func testAccImageAlias_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}

resource "incus_image" "img1" {
  project = incus_project.project1.name

  source_image = {
    remote = "images"
    name   = "alpine/edge"
    copy_aliases = false
  }
}

resource "incus_image_alias" "alias1" {
  alias          = "alpine-test"
  description    = "Alpine Linux"
  fingerprint    = incus_image.img1.fingerprint

  project        = incus_project.project1.name
  depends_on = [incus_image.img1]
}
	`, project)
}

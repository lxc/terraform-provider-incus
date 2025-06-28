package image_test

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccImage_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_basic(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_basicVM(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_basicVM(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1vm", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "source_image.type", "virtual-machine"),
					resource.TestCheckResourceAttr("incus_image.img1vm", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_alias(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_multipleAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img2", "alias.*", map[string]string{
						"name":        alias1,
						"description": alias1,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img2", "alias.*", map[string]string{
						"name":        alias2,
						"description": alias2,
					}),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_copiedAliases(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_copiedAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img3", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img3", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img3", "source_image.copy_aliases", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img3", "alias.*", map[string]string{
						"name": "alpine/edge",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img3", "alias.*", map[string]string{
						"name":        alias1,
						"description": alias1,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img3", "alias.*", map[string]string{
						"name":        alias2,
						"description": alias2,
					}),
					resource.TestCheckResourceAttr("incus_image.img3", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_aliasCollision(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_aliasCollision(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img4", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img4", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img4", "source_image.copy_aliases", "true"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img4", "alias.*", map[string]string{
						"name": "alpine/edge/amd64",
					}),
					resource.TestCheckResourceAttr("incus_image.img4", "copied_aliases.#", "4"),
				),
			},
		},
	})
}

func TestAccImage_aliasExists(t *testing.T) {
	alias := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_aliasExists1(alias),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image.copy_aliases", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.exists1", "alias.*", map[string]string{
						"name":        alias,
						"description": alias,
					}),
					resource.TestCheckResourceAttr("incus_image.exists1", "copied_aliases.#", "0"),
				),
			},
			{
				Config:      testAccImage_aliasExists2(alias),
				ExpectError: regexp.MustCompile(fmt.Sprintf(`Image alias %q already exists`, alias)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.exists1", "source_image.name", "alpine/edge"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.exists1", "alias.*", map[string]string{
						"name":        alias,
						"description": alias,
					}),
				),
			},
		},
	})
}

func TestAccImage_addRemoveAlias(t *testing.T) {
	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_alias(alias1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img2", "alias.*", map[string]string{
						"name":        alias1,
						"description": alias1,
					}),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccImage_multipleAliases(alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img2", "alias.*", map[string]string{
						"name":        alias1,
						"description": alias1,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img2", "alias.*", map[string]string{
						"name":        alias2,
						"description": alias2,
					}),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
			{
				Config: testAccImage_alias(alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img2", "source_image.copy_aliases", "false"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img2", "alias.*", map[string]string{
						"name":        alias2,
						"description": alias2,
					}),
					resource.TestCheckResourceAttr("incus_image.img2", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_project(t *testing.T) {
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_project(projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_instanceFromImageFingerprint(t *testing.T) {
	projectName := petname.Generate(2, "-")
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_instanceFromImageFingerprint(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_instance.inst", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.inst", "project", projectName),
				),
			},
		},
	})
}

func TestAccImage_architecture(t *testing.T) {
	projectName := petname.Name()
	architecture := "aarch64"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_architecture(projectName, architecture),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.architecture", architecture),
					resource.TestCheckResourceAttr("incus_image.img1", "project", projectName),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_oci(t *testing.T) {
	imageName := "alpine:latest"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_oci(imageName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.oci_img1", "source_image.remote", "docker"),
					resource.TestCheckResourceAttr("incus_image.oci_img1", "source_image.name", imageName),
				),
			},
		},
	})
}

func TestAccImage_sourceInstance(t *testing.T) {
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_sourceInstance(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1", "source_instance.name", instanceName),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img1", "alias.*", map[string]string{
						"name":        instanceName,
						"description": instanceName,
					}),
				),
			},
		},
	})
}

func TestAccImage_sourceInstanceWithSnapshot(t *testing.T) {
	projectName := petname.Name()
	instanceName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccImage_sourceInstanceWithSnapshot(projectName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1", "source_instance.name", instanceName),
					resource.TestCheckResourceAttr("incus_image.img1", "source_instance.snapshot", "snap0"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.img1", "alias.*", map[string]string{
						"name":        instanceName,
						"description": instanceName,
					}),
				),
			},
		},
	})
}

func TestAccImage_sourceFileSplitImage(t *testing.T) {
	tmpDir := t.TempDir()
	targetMetadata := filepath.Join(tmpDir, `alpine-edge.img`)
	targetData := targetMetadata + ".root"

	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckAPIExtensions(t, "image_create_aliases")
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {
				Source:            "null",
				VersionConstraint: ">= 3.0.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccSourceFileSplitImage_exportImage(targetMetadata),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.remote", "images"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.name", "alpine/edge"),
					resource.TestCheckResourceAttr("incus_image.img1", "source_image.copy_aliases", "true"),
					resource.TestCheckResourceAttr("incus_image.img1", "copied_aliases.#", "4"),
					resource.TestCheckResourceAttrSet("null_resource.export_img1", "id"),
				),
			},
			{
				Config: `#`, // Empty config to remove image. Comment is required, since empty string is seen as zero value.
			},
			{
				Config: testAccSourceFileSplitImage_fromFile(targetData, targetMetadata, alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.from_file", "source_file.data_path", targetData),
					resource.TestCheckResourceAttr("incus_image.from_file", "source_file.metadata_path", targetMetadata),
					resource.TestCheckResourceAttr("incus_image.from_file", "alias.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.from_file", "alias.*", map[string]string{
						"name": alias1,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.from_file", "alias.*", map[string]string{
						"name": alias2,
					}),
					resource.TestCheckResourceAttr("incus_image.from_file", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func TestAccImage_sourceFileUnifiedImage(t *testing.T) {
	name := petname.Generate(2, "-")
	tmpDir := t.TempDir()
	targetData := filepath.Join(tmpDir, name)

	alias1 := petname.Generate(2, "-")
	alias2 := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"null": {
				Source:            "null",
				VersionConstraint: ">= 3.0.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccSourceFileUnifiedImage_exportImage(name, targetData),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", name),
					resource.TestCheckResourceAttr("incus_instance.instance1", "image", "images:alpine/edge"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "type", "container"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttrSet("null_resource.publish_instance1", "id"),
					resource.TestCheckResourceAttrSet("null_resource.export_instance1_image", "id"),
					resource.TestCheckResourceAttrSet("null_resource.delete_instance1_image", "id"),
				),
			},
			{
				Config: testAccSourceFileUnifiedImage_fromFile(targetData, alias1, alias2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_image.from_file", "source_file.data_path", targetData+".tar.gz"),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.from_file", "alias.*", map[string]string{
						"name": alias1,
					}),
					resource.TestCheckTypeSetElemNestedAttrs("incus_image.from_file", "alias.*", map[string]string{
						"name": alias2,
					}),
					resource.TestCheckResourceAttr("incus_image.from_file", "copied_aliases.#", "0"),
				),
			},
		},
	})
}

func testAccImage_basic() string {
	return `
resource "incus_image" "img1" {
  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = true
  }
}
	`
}

func testAccImage_basicVM() string {
	return `
resource "incus_image" "img1vm" {
  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    type         = "virtual-machine"
    copy_aliases = true
  }
}
	`
}

func testAccImage_multipleAliases(alias1, alias2 string) string {
	return fmt.Sprintf(`
resource "incus_image" "img2" {
  alias {
    name = "%s"
    description = "%s"
  }

  alias {
    name = "%s"
    description = "%s"
  }

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = false
  }
}
	`, alias1, alias1, alias2, alias2)
}

func testAccImage_alias(alias string) string {
	return fmt.Sprintf(`
resource "incus_image" "img2" {
  alias {
    name = "%s"
    description = "%s"
  }

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = false
  }
}
	`, alias, alias)
}

func testAccImage_aliasExists1(alias string) string {
	return fmt.Sprintf(`
resource "incus_image" "exists1" {
  alias {
    name        = "%s"
	description = "%s"
  } 

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = false
  }
}
	`, alias, alias)
}

func testAccImage_aliasExists2(alias string) string {
	return fmt.Sprintf(`
resource "incus_image" "exists1" {
  alias {
    name        = "%s"
    description = "%s"
  }       

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = false
  }
}

resource "incus_image" "exists2" {
  alias {
    name        = "%s"
    description = "%s"
  }       

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = false
  }
}
	`, alias, alias, alias, alias)
}

func testAccImage_copiedAliases(alias1, alias2 string) string {
	return fmt.Sprintf(`
resource "incus_image" "img3" {
  alias {
    name = "alpine/edge"
  }

  alias {
    name        = "%s"
    description = "%s"
  }

  alias {
    name        = "%s"
    description = "%s"
  }

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = true
  }
}
	`, alias1, alias1, alias2, alias2)
}

func testAccImage_aliasCollision() string {
	return `
resource "incus_image" "img4" {
  alias {
    name = "alpine/edge/amd64"
  }

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = true
  }
}
	`
}

func testAccImage_project(project string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}

resource "incus_image" "img1" {
  project = incus_project.project1.name

  source_image = {
    remote = "images"
    name   = "alpine/edge"
  }
}
	`, project)
}

func testAccImage_instanceFromImageFingerprint(project string, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}

resource "incus_image" "img1" {
  project = incus_project.project1.name

  source_image = {
    remote = "images"
    name   = "alpine/edge"
  }
}

resource "incus_instance" "inst" {
  name    = "%s"
  project = incus_project.project1.name
  image   = incus_image.img1.fingerprint
  running = false

  device {
    name = "root"
    type = "disk"
    properties = {
      pool = "default"
      path = "/"
    }
  }
}
	`, project, instanceName)
}

func testAccImage_architecture(project string, architecture string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
}

resource "incus_image" "img1" {
  project = incus_project.project1.name

  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    architecture = "%s"
  }
}
	`, project, architecture)
}

func testAccImage_oci(image string) string {
	return fmt.Sprintf(`
resource "incus_image" "oci_img1" {
  source_image = {
    remote = "docker"
    name   = "%s"
  }
}
	`, image)
}

func testAccImage_sourceInstance(projectName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%[1]s"
  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  project = incus_project.project1.name
  name    = "%[2]s"
  image   = "%[3]s"
  running = false
}

resource "incus_image" "img1" {
  project = incus_project.project1.name

  alias {
    name        = incus_instance.instance1.name
    description = incus_instance.instance1.name
  }

  source_instance = {
    name = incus_instance.instance1.name
  }
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccImage_sourceInstanceWithSnapshot(projectName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%[1]s"
  config = {
    "features.images"   = false
    "features.profiles" = false
  }
}

resource "incus_instance" "instance1" {
  project = incus_project.project1.name
  name  = "%[2]s"
  image = "%[3]s"
}

resource "incus_instance_snapshot" "snapshot1" {
  project = incus_project.project1.name
  name     = "snap0"
  instance = incus_instance.instance1.name
  stateful = false
}

resource "incus_image" "img1" {
  project = incus_project.project1.name

  alias {
    name        = incus_instance.instance1.name
    description = incus_instance.instance1.name
  }

  source_instance = {
    name    = incus_instance.instance1.name
    snapshot = incus_instance_snapshot.snapshot1.name
  }
}
	`, projectName, instanceName, acctest.TestImage)
}

func testAccSourceFileSplitImage_exportImage(target string) string {
	return fmt.Sprintf(`
resource "incus_image" "img1" {
  source_image = {
    remote       = "images"
    name         = "alpine/edge"
    copy_aliases = true
  }
}

resource "null_resource" "export_img1" {
  provisioner "local-exec" {
    command = "incus image export ${incus_image.img1.fingerprint} %[1]s"
  }
}
`, target)
}

func testAccSourceFileSplitImage_fromFile(targetData, targetMetadata string, alias1, alias2 string) string {
	return fmt.Sprintf(`
resource "incus_image" "from_file" {
  source_file = {
    data_path     = "%[1]s"
    metadata_path = "%[2]s"
  }

  alias {
    name = "%[3]s"
  }

  alias {
    name = "%[4]s"
  }
}
`, targetData, targetMetadata, alias1, alias2)
}

func testAccSourceFileUnifiedImage_exportImage(name, targetData string) string {
	return fmt.Sprintf(`
resource "incus_instance" "instance1" {
  name    = "%[1]s"
  image   = "images:alpine/edge"
  type    = "container"
  running = false
}

resource "null_resource" "publish_instance1" {
  depends_on = [
    incus_instance.instance1
  ]
  provisioner "local-exec" {
    command = "incus publish --alias %[1]s %[1]s"
  }
}

resource "null_resource" "export_instance1_image" {
  depends_on = [
    null_resource.publish_instance1
  ]
  provisioner "local-exec" {
    command = "incus image export %[1]s %[2]s"
  }
}

resource "null_resource" "delete_instance1_image" {
  depends_on = [
    null_resource.export_instance1_image
  ]
  provisioner "local-exec" {
    command = "incus image delete %[1]s"
  }
}
`, name, targetData)
}

func testAccSourceFileUnifiedImage_fromFile(targetData string, alias1, alias2 string) string {
	return fmt.Sprintf(`
resource "incus_image" "from_file" {
  source_file = {
    data_path = "%[1]s.tar.gz"
  }

  alias {
    name = "%[2]s"
  }

  alias {
    name = "%[3]s"
  }
}

`, targetData, alias1, alias2)
}

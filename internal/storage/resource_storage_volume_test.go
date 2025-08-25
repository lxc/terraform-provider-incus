package storage_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccStorageVolume_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_basic(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "dir"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "type", "custom"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "filesystem"),
				),
			},
		},
	})
}

func TestAccStorageVolume_instanceAttach(t *testing.T) {
	instanceName := petname.Generate(2, "-")
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_instanceAttach(poolName, volumeName, instanceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "name", instanceName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "status", "Stopped"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.#", "1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.name", "volume1"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.type", "disk"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.path", "/mnt"),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.pool", poolName),
					resource.TestCheckResourceAttr("incus_instance.instance1", "device.0.properties.source", volumeName),
				),
			},
		},
	})
}

func TestAccStorageVolume_target(t *testing.T) {
	poolName := petname.Generate(2, "-")
	driverName := "dir"
	volumeName := petname.Generate(2, "-")

	clusterMemberNames := make(map[string]struct{}, 10) // It is unlikely, that acceptance tests are executed against clusters > 10 nodes.

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_target(poolName, driverName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					// This populates `clusterMemberNames` and therefore needs to be executed before the
					// check using this information.
					acctest.TestCheckGetClusterMemberNames(t, "data.incus_cluster.test", clusterMemberNames),

					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					acctest.TestCheckResourceAttrInLookup("incus_storage_volume.volume1", "target", clusterMemberNames),
				),
			},
		},
	})
}

func TestAccStorageVolume_project(t *testing.T) {
	volumeName := petname.Generate(2, "-")
	projectName := petname.Name()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_project(projectName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "project", projectName),
				),
			},
		},
	})
}

func TestAccStorageVolume_contentType(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_contentType(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "block"),
				),
			},
		},
	})
}

func TestAccStorageVolume_importBasic(t *testing.T) {
	volName := petname.Generate(2, "-")
	poolName := petname.Generate(2, "-")
	resourceName := "incus_storage_volume.volume1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_basic(poolName, volName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("/%s/%s", poolName, volName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccStorageVolume_importProject(t *testing.T) {
	volName := petname.Generate(2, "-")
	projectName := petname.Generate(2, "-")
	resourceName := "incus_storage_volume.volume1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_project(projectName, volName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/default/%s", projectName, volName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccStorageVolume_inheritedStoragePoolKeys(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_inheritedStoragePoolVolumeKeys(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "zfs"),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "config.volume.zfs.remove_snapshots", "true"),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "config.volume.zfs.use_refquota", "true"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "type", "custom"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "block"),

					// Ensure computed keys are not tracked.
					resource.TestCheckNoResourceAttr("incus_storage_volume.volume1", "config.zfs.remove_snapshots"),
					resource.TestCheckNoResourceAttr("incus_storage_volume.volume1", "config.zfs.use_refquota"),
				),
			},
		},
	})
}

func TestAccStorageVolume_sourceVolume(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_sourceVolume(poolName, volumeName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "name", fmt.Sprintf("%s-copy", volumeName)),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "source_volume.name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1_copy", "source_volume.pool", poolName),
				),
			},
			{
				Config: testAccStorageVolume_sourceVolume(poolName, volumeName),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccStorageVolume_sourceFileFilesystem(t *testing.T) {
	tmpDir := t.TempDir()
	backupFile := filepath.Join(tmpDir, "backup.tar.gz")

	poolName := petname.Generate(2, "-")
	sourceVolumeName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

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
				Config: testAccStorageVolume_sourceFileExportVolume(poolName, sourceVolumeName, backupFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", sourceVolumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "filesystem"),
				),
			},
			{
				Config: `#`, // Empty config to remove volume. Comment is required, since empty string is seen as zero value.
			},
			{
				Config: testAccStorageVolume_sourceFile(poolName, volumeName, backupFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "source_file", backupFile),
				),
			},
			{
				Config: testAccStorageVolume_sourceFile(poolName, volumeName, backupFile),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccStorageVolume_sourceFileBlock(t *testing.T) {
	tmpDir := t.TempDir()
	backupFile := filepath.Join(tmpDir, "backup.tar.gz")

	poolName := petname.Generate(2, "-")
	sourceVolumeName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

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
				Config: testAccStorageVolume_sourceFileExportBlockVolume(poolName, sourceVolumeName, backupFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", sourceVolumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "block"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "config.size", "1MiB"),
				),
			},
			{
				Config: `#`, // Empty config to remove volume. Comment is required, since empty string is seen as zero value.
			},
			{
				Config: testAccStorageVolume_sourceFile(poolName, volumeName, backupFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "source_file", backupFile),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "block"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "config.size", "1MiB"),
				),
			},
			{
				Config: testAccStorageVolume_sourceFile(poolName, volumeName, backupFile),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccStorageVolume_sourceFileIso(t *testing.T) {
	poolName := petname.Generate(2, "-")
	volumeName := petname.Generate(2, "-")

	tempDir := t.TempDir()
	isoFile := filepath.Join(tempDir, "image.iso")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)

			// Create ISO file with some null bytes
			data := make([]byte, 256)
			err := os.WriteFile(isoFile, data, 0o644)
			if err != nil {
				t.Fatal(err)
			}
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageVolume_sourceFile(poolName, volumeName, isoFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "lvm"),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "name", volumeName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "pool", poolName),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "source_file", isoFile),
					resource.TestCheckResourceAttr("incus_storage_volume.volume1", "content_type", "iso"),
				),
			},
			{
				Config: testAccStorageVolume_sourceFile(poolName, volumeName, isoFile),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccStorageVolume_basic(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "dir"
}

resource "incus_storage_volume" "volume1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
}
	`, poolName, volumeName)
}

func testAccStorageVolume_instanceAttach(poolName, volumeName, instanceName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
}

resource "incus_storage_volume" "volume1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
}

resource "incus_instance" "instance1" {
  name    = "%s"
  image   = "%s"
  running = false

  device {
    name = "volume1"
    type = "disk"
    properties = {
      path   = "/mnt"
      source = incus_storage_volume.volume1.name
      pool   = incus_storage_pool.pool1.name
    }
  }
}
	`, poolName, volumeName, instanceName, acctest.TestImage)
}

func testAccStorageVolume_target(poolName string, driverName string, volumeName string) string {
	return fmt.Sprintf(`
data "incus_cluster" "test" {}

locals {
  member_names = [ for k, v in data.incus_cluster.test.members : k ]
}

resource "incus_storage_pool" "storage_pool1_per_node" {
  // Unfortunately, the terraform plugin test framework does not support
  // "for_each", so we need to use "count" as an alternative.
  count = length(local.member_names)

  name   = "%[1]s"
  driver = "%[2]s"
  target = local.member_names[count.index]
}

resource "incus_storage_pool" "storage_pool1" {
  depends_on = [
    incus_storage_pool.storage_pool1_per_node,
  ]

  name   = "%[1]s"
  driver = "%[2]s"
}

resource "incus_storage_volume" "volume1" {
  name   = "%s"
  pool   = incus_storage_pool.storage_pool1.name
  target = local.member_names[0]
}
`, poolName, driverName, volumeName)
}

func testAccStorageVolume_project(projectName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name = "%s"
  config = {
    "features.storage.volumes" = false
  }
}

resource "incus_storage_volume" "volume1" {
  name    = "%s"
  pool    = "default"
  project = incus_project.project1.name
}
	`, projectName, volumeName)
}

func testAccStorageVolume_contentType(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "zfs"
}

resource "incus_storage_volume" "volume1" {
  name         = "%s"
  pool         = incus_storage_pool.pool1.name
  content_type = "block"
}
	`, poolName, volumeName)
}

func testAccStorageVolume_inheritedStoragePoolVolumeKeys(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver  = "zfs"
  config = {
    "volume.zfs.remove_snapshots" = "true",
	"volume.zfs.use_refquota" = "true"
  }
}

resource "incus_storage_volume" "volume1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
  content_type = "block"
  config = {
    "size" = "1GiB"
  }
}
	`, poolName, volumeName)
}

func testAccStorageVolume_sourceVolume(poolName, volumeName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "incus_storage_volume" "volume1" {
  name = "%[2]s"
  pool = incus_storage_pool.pool1.name

  config = {
    size = "1MiB"
  }
}

resource "incus_storage_volume" "volume1_copy" {
  name = "%[2]s-copy"
  pool = "default"

  source_volume = {
    pool = incus_storage_pool.pool1.name
    name = incus_storage_volume.volume1.name
  }
}
`,
		poolName, volumeName)
}

func testAccStorageVolume_sourceFileExportVolume(poolName, volumeName, backupFile string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "incus_storage_volume" "volume1" {
  name = "%[2]s"
  pool = incus_storage_pool.pool1.name
}

resource "null_resource" "export_volume1" {
  provisioner "local-exec" {
    command = "incus storage volume export %[1]s ${incus_storage_volume.volume1.name} %[3]s"
  }
}
`,
		poolName, volumeName, backupFile)
}

func testAccStorageVolume_sourceFileExportBlockVolume(poolName, volumeName, backupFile string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "incus_storage_volume" "volume1" {
  name = "%[2]s"
  pool = incus_storage_pool.pool1.name

  content_type = "block"

  config = {
    size = "1MiB"
  }
}

resource "null_resource" "export_volume1" {
  provisioner "local-exec" {
    command = "incus storage volume export %[1]s ${incus_storage_volume.volume1.name} %[3]s"
  }
}
`,
		poolName, volumeName, backupFile)
}

func testAccStorageVolume_sourceFile(poolName, volumeName, sourceFile string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%[1]s"
  driver = "lvm"
}

resource "incus_storage_volume" "volume1" {
  name = "%[2]s"
  pool = incus_storage_pool.pool1.name
  source_file = "%[3]s"
}
`,
		poolName, volumeName, sourceFile)
}

package storage_test

import (
	"fmt"
	"path/filepath"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccStorageBucket_basic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_basic(poolName, bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "name", poolName),
					resource.TestCheckResourceAttr("incus_storage_pool.pool1", "driver", "dir"),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", poolName),
				),
			},
		},
	})
}

func TestAccStorageBucket_target(t *testing.T) {
	poolName := petname.Generate(2, "-")
	driverName := "dir"
	bucketName := petname.Generate(2, "-")

	clusterMemberNames := make(map[string]struct{}, 10) // It is unlikely, that acceptance tests are executed against clusters > 10 nodes.

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_target(poolName, driverName, bucketName),
				Check: resource.ComposeTestCheckFunc(
					// This populates `clusterMemberNames` and therefore needs to be executed before the
					// check using this information.
					acctest.TestCheckGetClusterMemberNames(t, "data.incus_cluster.test", clusterMemberNames),

					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", poolName),
					acctest.TestCheckResourceAttrInLookup("incus_storage_bucket.bucket1", "target", clusterMemberNames),
				),
			},
		},
	})
}

func TestAccStorageBucket_project(t *testing.T) {
	projectName := petname.Name()
	bucketName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_project(projectName, bucketName),
				Check: resource.ComposeTestCheckFunc(

					resource.TestCheckResourceAttr("incus_project.project1", "name", projectName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "project", projectName),
				),
			},
		},
	})
}

func TestAccStorageBucket_importBasic(t *testing.T) {
	poolName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	resourceName := "incus_storage_bucket.bucket1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_basic(poolName, bucketName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("/%s/%s", poolName, bucketName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportState:                          true,
				ImportStateVerify:                    true,
			},
		},
	})
}

func TestAccStorageBucket_importProject(t *testing.T) {
	projectName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")
	resourceName := "incus_storage_bucket.bucket1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_project(projectName, bucketName),
			},
			{
				ResourceName:                         resourceName,
				ImportStateId:                        fmt.Sprintf("%s/default/%s", projectName, bucketName),
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerify:                    true,
				ImportState:                          true,
			},
		},
	})
}

func TestAccStorageBucket_sourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	backupFile := filepath.Join(tmpDir, "backup.tar.gz")

	sourceBucketName := petname.Generate(2, "-")
	bucketName := petname.Generate(2, "-")

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
				Config: testAccStorageBucket_sourceFileExportBucket(sourceBucketName, backupFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", sourceBucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "description", "Some description"),
				),
			},
			{
				Config: `#`, // Empty config to remove bucket. Comment is required, since empty string is seen as zero value.
			},
			{
				Config: testAccStorageBucket_sourceFile(bucketName, backupFile),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", "default"),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "source_file", backupFile),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "description", "Some description"),
				),
			},
		},
	})
}

func testAccStorageBucket_basic(poolName string, bucketName string) string {
	return fmt.Sprintf(`
resource "incus_storage_pool" "pool1" {
  name   = "%s"
  driver = "dir"
}

resource "incus_storage_bucket" "bucket1" {
  name = "%s"
  pool = incus_storage_pool.pool1.name
}
	`, poolName, bucketName)
}

func testAccStorageBucket_target(poolName string, driverName, bucketName string) string {
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

resource "incus_storage_bucket" "bucket1" {
  name   = "%[3]s"
  pool   = incus_storage_pool.storage_pool1.name
  target = local.member_names[0]
}
`, poolName, driverName, bucketName)
}

func testAccStorageBucket_project(projectName string, bucketName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
	name = "%s"
	config = {
		"features.storage.volumes" = false
	}
}

resource "incus_storage_bucket" "bucket1" {
	name    = "%s"
	pool    = "default"
	project = incus_project.project1.name
}
	`, projectName, bucketName)
}

func testAccStorageBucket_sourceFileExportBucket(bucketName string, backupFile string) string {
	return fmt.Sprintf(`
resource "incus_storage_bucket" "bucket1" {
	name        = "%s"
	pool        = "default"
	description = "Some description"
}

resource "null_resource" "export_bucket1" {
  provisioner "local-exec" {
    command = "incus storage bucket export default ${incus_storage_bucket.bucket1.name} %s"
  }
}
	`, bucketName, backupFile)
}

func testAccStorageBucket_sourceFile(bucketName string, sourceFile string) string {
	return fmt.Sprintf(`
resource "incus_storage_bucket" "bucket1" {
	name        = "%s"
	pool        = "default"
	source_file = "%s"
}
	`, bucketName, sourceFile)
}

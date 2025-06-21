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
	bucketName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccStorageBucket_target(bucketName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "name", bucketName),
					resource.TestCheckResourceAttr("incus_storage_bucket.bucket1", "pool", "default"),
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

func testAccStorageBucket_target(bucketName string) string {
	return fmt.Sprintf(`
resource "incus_storage_bucket" "bucket1" {
	name    = "%s"
	pool    = "default"
	target = "node-2"
}
 	`, bucketName)
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

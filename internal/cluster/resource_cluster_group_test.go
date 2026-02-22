package cluster_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccClusterGroup_basic(t *testing.T) {
	clusterGroupName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterGroup_basic(clusterGroupName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_cluster_group.group1", "name", clusterGroupName),
					resource.TestCheckResourceAttr("incus_cluster_group.group1", "description", ""),
				),
			},
		},
	})
}

func TestAccClusterGroup_description(t *testing.T) {
	clusterGroupName := petname.Generate(2, "-")
	description := petname.Adjective()

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterGroup_description(clusterGroupName, description),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_cluster_group.group1", "name", clusterGroupName),
					resource.TestCheckResourceAttr("incus_cluster_group.group1", "description", description),
				),
			},
		},
	})
}

func TestAccClusterGroup_withMembers(t *testing.T) {
	clusterGroupName := petname.Generate(2, "-")
	clusterMemberNames := make(map[string]struct{}, 10) // It is unlikely, that acceptance tests are executed against clusters > 10 nodes.

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccClusterGroup_withMembers(clusterGroupName),
				Check: resource.ComposeTestCheckFunc(
					// This populates `clusterMemberNames` and therefore needs to be executed before the
					// check using this information.
					acctest.TestCheckGetClusterMemberNames(t, "data.incus_cluster.test", clusterMemberNames),

					resource.TestCheckResourceAttr("incus_cluster_group.group1", "name", clusterGroupName),
					resource.TestCheckResourceAttr("incus_cluster_group.group1", "members.#", "1"),
				),
			},
		},
	})
}

func testAccClusterGroup_basic(name string) string {
	return fmt.Sprintf(`
resource "incus_cluster_group" "group1" {
  name   = "%s"
}
`, name)
}

func testAccClusterGroup_description(name string, description string) string {
	return fmt.Sprintf(`
resource "incus_cluster_group" "group1" {
  name        = "%s"
  description = "%s"
}
`, name, description)
}

func testAccClusterGroup_withMembers(name string) string {
	return fmt.Sprintf(`
data "incus_cluster" "test" {}

locals {
  member_names = [ for k, v in data.incus_cluster.test.members : k ]
}

resource "incus_cluster_group" "group1" {
  name    = "%s"
  members = [local.member_names[0]]
}
`, name)
}

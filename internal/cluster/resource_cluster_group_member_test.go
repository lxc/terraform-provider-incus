package cluster_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccClusterGroupMember_basic(t *testing.T) {
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
				Config: testAccClusterGroupMember_basic(clusterGroupName),
				Check: resource.ComposeTestCheckFunc(
					// This populates `clusterMemberNames` and therefore needs to be executed before the
					// check using this information.
					acctest.TestCheckGetClusterMemberNames(t, "data.incus_cluster.test", clusterMemberNames),

					resource.TestCheckResourceAttr("incus_cluster_group_member.group1_node1", "cluster_group", clusterGroupName),
					acctest.TestCheckResourceAttrInLookup("incus_cluster_group_member.group1_node1", "member", clusterMemberNames),
				),
			},
		},
	})
}

func testAccClusterGroupMember_basic(clusterGroupName string) string {
	return fmt.Sprintf(`
data "incus_cluster" "test" {}

locals {
  member_names = [ for k, v in data.incus_cluster.test.members : k ]
}

resource "incus_cluster_group" "group1" {
  name   = "%[1]s"
}

resource "incus_cluster_group_member" "group1_node1" {
  cluster_group = incus_cluster_group.group1.name
  member        = local.member_names[0]
}
`, clusterGroupName)
}

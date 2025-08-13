package clustering_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccClusterDataSource_Standalone(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckStandalone(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "incus_cluster" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.incus_cluster.test", "is_clustered", "false"),
					resource.TestCheckResourceAttr("data.incus_cluster.test", "members.%", "1"),
					// Since the name of the cluster member is the empty string in the
					// non-clustered case, there are two consecutive dots to address the properties.
					resource.TestCheckResourceAttr("data.incus_cluster.test", "members..%", "7"),                                                       // Expected attributes per cluster member.
					resource.TestMatchResourceAttr("data.incus_cluster.test", "members..address", regexp.MustCompile(`(^https://.+:[1-9][0-9]*$|^$)`)), // Expect valid URL or empty string (if Incus is not bound to network and only controlled via unix socket).
					// Architecture list from https://linuxcontainers.org/incus/docs/main/architectures/
					resource.TestMatchResourceAttr("data.incus_cluster.test", "members..architecture", regexp.MustCompile(`i686|x86_64|armv6l|armv7l|armv8l|aarch64|ppc|ppc64|ppc64le|s390x|mips|mips64|riscv32|riscv64|loongarch64`)), // Expect architecture to have a supported value
					resource.TestCheckResourceAttr("data.incus_cluster.test", "members..failure_domain", ""),                                                                                                                           // Failure domain is undefined for standalone instances.
					resource.TestCheckResourceAttr("data.incus_cluster.test", "members..groups.#", "0"),                                                                                                                                // Cluster member groups are undefined for standalone instances.
					resource.TestCheckResourceAttr("data.incus_cluster.test", "members..roles.#", "0"),                                                                                                                                 // Cluster member roles are undefined for standalone instances.
					resource.TestCheckResourceAttr("data.incus_cluster.test", "members..status", "Online"),                                                                                                                             // Standalone instances are always considered online, if fetch data is successful.
				),
			},
		},
	})
}

func TestAccClusterDataSource_Clustering(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckClustering(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "incus_cluster" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.incus_cluster.test", "is_clustered", "true"),
					resource.TestMatchResourceAttr("data.incus_cluster.test", "members.%", regexp.MustCompile(`^[1-9][0-9]*$`)), // Expect 1 or more cluster members.
					resource.TestMatchTypeSetElemNestedAttrs("data.incus_cluster.test", "members.*", map[string]*regexp.Regexp{
						"%":       regexp.MustCompile(`7`),                        // Expected attributes per cluster member.
						"address": regexp.MustCompile(`^https://.+:[1-9][0-9]*$`), // Expect valid URL.
						// Architecture list from https://linuxcontainers.org/incus/docs/main/architectures/
						"architecture":   regexp.MustCompile(`i686|x86_64|armv6l|armv7l|armv8l|aarch64|ppc|ppc64|ppc64le|s390x|mips|mips64|riscv32|riscv64|loongarch64`), // Expect architecture to have a supported value.
						"failure_domain": regexp.MustCompile(`^.+$`),                                                                                                     // Expect non empty string.
						"groups.#":       regexp.MustCompile(`^[1-9][0-9]*$`),                                                                                            // Expect at least 1 group.
						"roles.#":        regexp.MustCompile(`^[1-9][0-9]*$`),                                                                                            // Expect at least 1 role.
						// Status list from https://github.com/lxc/incus/blob/40095bbc23512120e3a3be6d74d94ccb490e4e8e/internal/server/db/node.go#L148-L174
						"status": regexp.MustCompile(`^Online|Evacuated|Offline|Blocked$`), // Expect status to have a supported value.
					}),
				),
			},
		},
	})
}

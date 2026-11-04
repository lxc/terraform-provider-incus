package image_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccImageDataSource_ImageFingerprintNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "incus_image" "test" {
  fingerprint = "0000000000000000000000000000000000000000000000000000000000000000"
}
`,
				ExpectError: regexp.MustCompile(`Failed to retrieve image by fingerprint`),
			},
		},
	})
}

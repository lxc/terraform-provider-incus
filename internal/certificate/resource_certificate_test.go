package certificate_test

import (
	"fmt"
	"testing"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/lxc/terraform-provider-incus/internal/acctest"
)

func TestAccCertificate_basic(t *testing.T) {
	certificateName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificate_basic(certificateName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_certificate.cert1", "name", certificateName),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "description", ""),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "type", "client"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "restricted", "false"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "projects.#", "0"),
				),
			},
		},
	})
}

func TestAccCertificate_withProject(t *testing.T) {
	certificateName := petname.Generate(2, "-")
	projectName := petname.Generate(1, "")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificate_withProject(certificateName, projectName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_certificate.cert1", "name", certificateName),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "description", ""),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "type", "metrics"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "restricted", "true"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "projects.#", "1"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "projects.0", projectName),
				),
			},
		},
	})
}

func TestAccCertificate_withWhitespaces(t *testing.T) {
	certificateName := petname.Generate(2, "-")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
		},
		ProtoV6ProviderFactories: acctest.ProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCertificate_withWhitespaces(certificateName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("incus_certificate.cert1", "name", certificateName),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "description", ""),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "type", "client"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "restricted", "false"),
					resource.TestCheckResourceAttr("incus_certificate.cert1", "projects.#", "0"),
				),
			},
		},
	})
}

func testAccCertificate_basic(name string) string {
	return fmt.Sprintf(`
resource "incus_certificate" "cert1" {
  name        = "%s"
  certificate = "-----BEGIN CERTIFICATE-----\nMIIBwjCCAUigAwIBAgIUCGycHG038IvNWOBtciK4Bk7fB3wwCgYIKoZIzj0EAwMw\nGDEWMBQGA1UEAwwNbWV0cmljcy5sb2NhbDAeFw0yNDExMDUxNzU2MDdaFw0zNDEx\nMDMxNzU2MDdaMBgxFjAUBgNVBAMMDW1ldHJpY3MubG9jYWwwdjAQBgcqhkjOPQIB\nBgUrgQQAIgNiAASJeWxvoByh7+4A6k+SrrpQ/NGBRPvqBloV5fTmy9uPaRMZew9K\nIVg/8+7ciXK4193eLeVBQiILxj++a5lCvthmJcbpRkckyXuhQc4/JMuTW2h6jYWX\nTsTZfJEnvYU4IpqjUzBRMB0GA1UdDgQWBBQAqliKxB7id1A+4TQU0adTAB0+RTAf\nBgNVHSMEGDAWgBQAqliKxB7id1A+4TQU0adTAB0+RTAPBgNVHRMBAf8EBTADAQH/\nMAoGCCqGSM49BAMDA2gAMGUCMFYzGT/0ko01qFrD8QFkqhNPzuSA6yV8p6SSKUk2\nJ/35p8EoEmVb1LWldJ4KOxu8nAIxAOkoWTOfi0Nrb4MeKyu1R2zqD+CfgUlZjhLi\n4+1L464g/5a/nSIfDX+VyC+PNGBQFw==\n-----END CERTIFICATE-----\n"
}
`, name)
}

func testAccCertificate_withProject(name string, projectName string) string {
	return fmt.Sprintf(`
resource "incus_project" "project1" {
  name        = "%[1]s"
}

resource "incus_certificate" "cert1" {
  name        = "%[2]s"
  restricted  = true
  type        = "metrics"
  projects    = [incus_project.project1.name]
  certificate = "-----BEGIN CERTIFICATE-----\nMIIBwjCCAUigAwIBAgIUCGycHG038IvNWOBtciK4Bk7fB3wwCgYIKoZIzj0EAwMw\nGDEWMBQGA1UEAwwNbWV0cmljcy5sb2NhbDAeFw0yNDExMDUxNzU2MDdaFw0zNDEx\nMDMxNzU2MDdaMBgxFjAUBgNVBAMMDW1ldHJpY3MubG9jYWwwdjAQBgcqhkjOPQIB\nBgUrgQQAIgNiAASJeWxvoByh7+4A6k+SrrpQ/NGBRPvqBloV5fTmy9uPaRMZew9K\nIVg/8+7ciXK4193eLeVBQiILxj++a5lCvthmJcbpRkckyXuhQc4/JMuTW2h6jYWX\nTsTZfJEnvYU4IpqjUzBRMB0GA1UdDgQWBBQAqliKxB7id1A+4TQU0adTAB0+RTAf\nBgNVHSMEGDAWgBQAqliKxB7id1A+4TQU0adTAB0+RTAPBgNVHRMBAf8EBTADAQH/\nMAoGCCqGSM49BAMDA2gAMGUCMFYzGT/0ko01qFrD8QFkqhNPzuSA6yV8p6SSKUk2\nJ/35p8EoEmVb1LWldJ4KOxu8nAIxAOkoWTOfi0Nrb4MeKyu1R2zqD+CfgUlZjhLi\n4+1L464g/5a/nSIfDX+VyC+PNGBQFw==\n-----END CERTIFICATE-----\n"
}
`, projectName, name)
}

func testAccCertificate_withWhitespaces(name string) string {
	return fmt.Sprintf(`
resource "incus_certificate" "cert1" {
  name        = "%s"
  certificate = <<EOT
-----BEGIN CERTIFICATE-----
MIIB0TCCAVagAwIBAgIQdcncrUO5YxyVEZtetEsKdzAKBggqhkjOPQQDAzAwMRkw
FwYDVQQKExBMaW51eCBDb250YWluZXJzMRMwEQYDVQQDDApyb290QGhwODQwMB4X
DTI1MTEyNjA5MzM1OVoXDTM1MTEyNDA5MzM1OVowMDEZMBcGA1UEChMQTGludXgg
Q29udGFpbmVyczETMBEGA1UEAwwKcm9vdEBocDg0MDB2MBAGByqGSM49AgEGBSuB
BAAiA2IABOkwSJ1gvUUDMnYAXKSVKTt0NVkdpjrRmCrD5zEKDeWO1yNDAHDxzGq3
422mEJWi0vw28Jf9HNHj758a6AGhkwh4iUB1VfP80vXJIFjMXA38K31OYtnQqres
Q84sWKlX2qM1MDMwDgYDVR0PAQH/BAQDAgWgMBMGA1UdJQQMMAoGCCsGAQUFBwMC
MAwGA1UdEwEB/wQCMAAwCgYIKoZIzj0EAwMDaQAwZgIxAMHlzDBhms+LFAC/ZMVh
nFuZQHi8u/XYnlYHKNjiygvDRSsxM1fM1yfi42D0Iq8h5QIxAKPzZ/W6eRae0BbP
jL2lT/FPHfxsvOc+Orf34EX5aCjAfUDxBi1jIxYNum+v2JHhQQ==
-----END CERTIFICATE-----

EOT
}
`, name)
}

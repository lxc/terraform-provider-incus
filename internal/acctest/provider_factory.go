package acctest

import (
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	incus_config "github.com/lxc/incus/v6/shared/cliconfig"

	"github.com/lxc/terraform-provider-incus/internal/provider"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

// TestImage is a constant that specifies the default image used in all tests.
var TestImage = "images:alpine/edge/default/amd64"

// Updating TestImage architecture is best effort, on error just keep the default value.
func init() {
	p := testProvider()
	server, err := p.InstanceServer("", "", "")
	if err != nil {
		return
	}

	apiServer, _, err := server.GetServer()
	if err != nil {
		return
	}

	if apiServer.Environment.KernelArchitecture == "x86_64" {
		// Short circuit, default test image is already a fit.
		return
	}

	imageServer, err := p.ImageServer("images")
	if err != nil {
		return
	}

	images, err := imageServer.GetImages()
	if err != nil {
		return
	}

	for _, image := range images {
		if image.Architecture != apiServer.Environment.KernelArchitecture {
			continue
		}

		for _, alias := range image.Aliases {
			if strings.HasPrefix(alias.Name, "alpine/edge/default/") {
				TestImage = "images:" + alias.Name
				return
			}
		}
	}
}

var (
	testProviderConfig *provider_config.IncusProviderConfig
	testProviderMutex  sync.Mutex
)

// testProvider returns an IncusProviderConfig that is initialized with default
// Incus config.
//
// NOTE: This means this provider can differ from the actual provider used
// within the test. Therefore, it should be used exclusively for test prechecks
// because we assume all tests are run locally.
func testProvider() *provider_config.IncusProviderConfig {
	testProviderMutex.Lock()
	defer testProviderMutex.Unlock()

	if testProviderConfig == nil {
		var config *incus_config.Config

		incusRemote := os.Getenv("INCUS_REMOTE")
		if incusRemote != "" {
			var err error
			config, err = incus_config.LoadConfig("")
			if err != nil {
				panic(err)
			}

			config.DefaultRemote = incusRemote
		} else {
			config = incus_config.DefaultConfig()
		}

		acceptClientCert := true
		testProviderConfig = provider_config.NewIncusProvider(config, acceptClientCert)
	}

	return testProviderConfig
}

// ProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"incus": providerserver.NewProtocol6WithError(provider.NewIncusProvider("test")()),
}

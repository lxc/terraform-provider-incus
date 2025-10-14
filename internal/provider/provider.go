package provider

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus_config "github.com/lxc/incus/v6/shared/cliconfig"
	incus_shared "github.com/lxc/incus/v6/shared/util"

	"github.com/lxc/terraform-provider-incus/internal/certificate"
	"github.com/lxc/terraform-provider-incus/internal/cluster"
	"github.com/lxc/terraform-provider-incus/internal/image"
	"github.com/lxc/terraform-provider-incus/internal/instance"
	"github.com/lxc/terraform-provider-incus/internal/network"
	"github.com/lxc/terraform-provider-incus/internal/profile"
	"github.com/lxc/terraform-provider-incus/internal/project"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
	provider_validators "github.com/lxc/terraform-provider-incus/internal/provider-validators"
	"github.com/lxc/terraform-provider-incus/internal/server"
	"github.com/lxc/terraform-provider-incus/internal/storage"
)

// IncusProviderRemoteModel represents provider's schema remote.
type IncusProviderRemoteModel struct {
	Name               types.String `tfsdk:"name"`
	Address            types.String `tfsdk:"address"`
	Protocol           types.String `tfsdk:"protocol"`
	AuthenticationType types.String `tfsdk:"authentication_type"`
	Token              types.String `tfsdk:"token"`
	Public             types.Bool   `tfsdk:"public"`
}

// IncusProviderModel represents provider's schema.
type IncusProviderModel struct {
	Remotes                    []IncusProviderRemoteModel `tfsdk:"remote"`
	DefaultRemote              types.String               `tfsdk:"default_remote"`
	ConfigDir                  types.String               `tfsdk:"config_dir"`
	Project                    types.String               `tfsdk:"project"`
	AcceptRemoteCertificate    types.Bool                 `tfsdk:"accept_remote_certificate"`
	GenerateClientCertificates types.Bool                 `tfsdk:"generate_client_certificates"`
}

// IncusProvider ...
type IncusProvider struct {
	version string
}

// New returns Incus provider with the given version set.
func NewIncusProvider(version string) func() provider.Provider {
	return func() provider.Provider {
		return &IncusProvider{
			version: version,
		}
	}
}

func (p *IncusProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "incus"
	resp.Version = p.version
}

func (p *IncusProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"config_dir": schema.StringAttribute{
				Optional:    true,
				Description: "The directory to look for existing Incus configuration. (default = $HOME/.config/incus)",
			},

			"generate_client_certificates": schema.BoolAttribute{
				Optional:    true,
				Description: "Automatically generate the Incus client certificates if they don't exist.",
			},

			"accept_remote_certificate": schema.BoolAttribute{
				Optional:    true,
				Description: "Accept the server certificate.",
			},

			"project": schema.StringAttribute{
				Optional:    true,
				Description: "The project where project-scoped resources will be created. Can be overridden in individual resources. (default = default)",
			},
			"default_remote": schema.StringAttribute{
				Optional:    true,
				Description: "The default remote to use when no other remote is defined in a resource.",
			},
		},

		Blocks: map[string]schema.Block{
			"remote": schema.ListNestedBlock{
				Description: "Incus Remote",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the Incus remote.",
						},

						"address": schema.StringAttribute{
							Optional:    true,
							Description: "The URL of the Incus host. The default will be the path to the local unix socket, or leaving it as an empty string will use the default socket path.",
						},

						"protocol": schema.StringAttribute{
							Optional:    true,
							Description: "Server protocol ( incus, oci or simplestreams )",
							Validators: []validator.String{
								stringvalidator.OneOf("incus", "oci", "simplestreams"),
							},
						},

						"public": schema.BoolAttribute{
							Optional:    true,
							Description: "Public image server",
						},

						"authentication_type": schema.StringAttribute{
							Optional:    true,
							Description: "Server authentication type, tls or oidc. ( Only for the `incus` protocol )",
							Validators: []validator.String{
								stringvalidator.OneOf("tls", "oidc"),
								provider_validators.CheckProtocol("incus"),
							},
						},

						"token": schema.StringAttribute{
							Optional:    true,
							Sensitive:   true,
							Description: "The trust token used for initial authentication with the Incus remote.",
						},
					},
				},
			},
		},
	}
}

func (p *IncusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data IncusProviderModel

	// Read provider schema into model.
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	// Determine Incus configuration directory.
	configDir := data.ConfigDir.ValueString()
	if configDir == "" {
		configDir = "$HOME/.config/incus"

		v, ok := os.LookupEnv("INCUS_CONF")
		if ok {
			configDir = v
		}
	}
	configDir = os.ExpandEnv(configDir)

	// Try to load config.yml from determined configDir. If there's
	// an error loading config.yml, default config will be used.
	configPath := filepath.Join(configDir, "config.yml")
	config, err := incus_config.LoadConfig(configPath)
	if err != nil {
		config = incus_config.DefaultConfig()
		config.ConfigDir = configDir
	}

	log.Printf("[DEBUG] Incus Config: %#v", config)

	// Determine if the Incus server's SSL certificates should be
	// accepted. If this is set to false and if the remote's
	// certificates haven't already been accepted, the user will
	// need to accept the certificates out of band of Terraform.
	acceptServerCertificate := data.AcceptRemoteCertificate.ValueBool()
	if data.AcceptRemoteCertificate.IsNull() || data.AcceptRemoteCertificate.IsUnknown() {
		v, ok := os.LookupEnv("INCUS_ACCEPT_SERVER_CERTIFICATE")
		if ok {
			acceptServerCertificate = incus_shared.IsTrue(v)
		}
	}

	// Determine if the client Incus (ie: the workstation running Terraform)
	// should generate client certificates if they don't already exist.
	generateClientCertificates := data.GenerateClientCertificates.ValueBool()
	if data.AcceptRemoteCertificate.IsNull() || data.GenerateClientCertificates.IsUnknown() {
		v, ok := os.LookupEnv("INCUS_GENERATE_CLIENT_CERTS")
		if ok {
			generateClientCertificates = incus_shared.IsTrue(v)
		}
	}

	if generateClientCertificates {
		err := config.GenerateClientCertificate()
		if err != nil {
			resp.Diagnostics.AddError("Failed to generate client certificate", err.Error())
			return
		}
	}

	// Determine project.
	project := data.Project.ValueString()
	if project != "" {
		config.ProjectOverride = project
	}

	// Initialize global IncusProvider struct.
	// This struct is used to store information about this Terraform
	// provider's configuration for reference throughout the lifecycle.
	incusProvider := provider_config.NewIncusProvider(config, acceptServerCertificate)

	// Create Incus remote from environment variables (if defined).
	// This emulates the Terraform provider "remote" config:
	//
	// remote {
	//   name    = INCUS_REMOTE
	//   address = INCUS_ADDR
	//   ...
	// }
	envName := os.Getenv("INCUS_REMOTE")
	if envName != "" {
		incusAddress := os.Getenv("INCUS_ADDR")

		environmentRemote := provider_config.IncusProviderRemoteConfig{
			Name:               envName,
			Address:            incusAddress,
			Protocol:           os.Getenv("INCUS_PROTOCOL"),
			AuthenticationType: os.Getenv("INCUS_AUTHENTICATION_TYPE"),
			Token:              os.Getenv("INCUS_TOKEN"),
			Public:             false,
		}

		// This will be the default remote unless overridden by an
		// explicitly defined remote in the Terraform configuration.
		incusProvider.SetRemote(environmentRemote, true)
	}

	// Loop over Incus Remotes defined in the schema and create
	// an IncusProviderRemoteConfig for each one.
	//
	// This does not yet connect to any of the defined remotes,
	// it only stores the configuration information until it is
	// necessary to connect to the remote.
	//
	// This lazy loading allows this Incus provider to be used
	// in Terraform configurations where the Incus remote might not
	// exist yet.
	for _, remote := range data.Remotes {
		isDefault := false

		protocol := remote.Protocol.ValueString()
		if protocol == "" {
			protocol = "incus"
		}

		autheticationType := remote.AuthenticationType.ValueString()
		if autheticationType == "" {
			autheticationType = "tls"
		}

		incusProviderRemoteConfig := provider_config.IncusProviderRemoteConfig{
			Name:               remote.Name.ValueString(),
			Address:            remote.Address.ValueString(),
			Protocol:           protocol,
			AuthenticationType: autheticationType,
			Token:              remote.Token.ValueString(),
			Public:             remote.Public.ValueBool(),
		}

		if data.DefaultRemote.ValueString() == remote.Name.ValueString() {
			isDefault = true
		}
		incusProvider.SetRemote(incusProviderRemoteConfig, isDefault)
	}

	log.Printf("[DEBUG] Incus Provider: %#v", &incusProvider)

	resp.ResourceData = incusProvider
	resp.DataSourceData = incusProvider
}

func (p *IncusProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		certificate.NewCertificateResource,
		cluster.NewClusterGroupMemberResource,
		cluster.NewClusterGroupResource,
		image.NewImageResource,
		instance.NewInstanceResource,
		instance.NewInstanceSnapshotResource,
		network.NewNetworkACLResource,
		network.NewNetworkForwardResource,
		network.NewNetworkAddressSet,
		network.NewNetworkIntegrationResource,
		network.NewNetworkLBResource,
		network.NewNetworkPeerResource,
		network.NewNetworkResource,
		network.NewNetworkZoneRecordResource,
		network.NewNetworkZoneResource,
		profile.NewProfileResource,
		project.NewProjectResource,
		server.NewServerResource,
		storage.NewStorageBucketKeyResource,
		storage.NewStorageBucketResource,
		storage.NewStoragePoolResource,
		storage.NewStorageVolumeResource,
	}
}

func (p *IncusProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	dataSources := []func() datasource.DataSource{
		cluster.NewClusterDataSource,
		image.NewImageDataSource,
	}

	dataSources = append(dataSources, generatedDataSources()...)

	return dataSources
}

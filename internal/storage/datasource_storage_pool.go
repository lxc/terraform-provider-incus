package storage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type StoragePoolDataSource struct {
	provider *provider_config.IncusProviderConfig
}

func NewStoragePoolDataSource() datasource.DataSource {
	return &StoragePoolDataSource{}
}

func (d *StoragePoolDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_pool", req.ProviderTypeName)
}

func (d *StoragePoolDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},

			"driver": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},

			"project": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			"target": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"config": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *StoragePoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.IncusProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	d.provider = provider
}

func (d *StoragePoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state StoragePoolModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	projectName := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := d.provider.InstanceServer(remote, projectName, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	storagePoolName := state.Name.ValueString()
	storagePool, _, err := server.GetStoragePool(storagePoolName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage pool %q", storagePoolName), err.Error())
		return
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(storagePool.Config), state.Config)

	state.Name = types.StringValue(storagePool.Name)
	state.Description = types.StringValue(storagePool.Description)
	state.Driver = types.StringValue(storagePool.Driver)
	state.Config = config

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

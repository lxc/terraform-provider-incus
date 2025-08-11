package server

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type ServerDataSourceModel struct {
	Addresses       types.List   `tfsdk:"addresses"`
	Remote          types.String `tfsdk:"remote"`
	ServerClustered types.Bool   `tfsdk:"server_clustered"`
	ServerName      types.String `tfsdk:"server_name"`
	Target          types.String `tfsdk:"target"`
}

type ServerDataSource struct {
	provider *provider_config.IncusProviderConfig
}

func NewServerDataSource() datasource.DataSource {
	return &ServerDataSource{}
}

func (d *ServerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_server", req.ProviderTypeName)
}

func (d *ServerDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"addresses": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},

			"server_clustered": schema.BoolAttribute{
				Computed: true,
			},

			"server_name": schema.StringAttribute{
				Computed: true,
			},

			"target": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (d *ServerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ServerDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	target := state.Target.ValueString()

	serverProvider, err := d.provider.InstanceServer(remote, "", target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// get server environment.
	server, _, err := serverProvider.GetServer()
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve server environment", err.Error())
		return
	}

	serverAddresses, diags := types.ListValueFrom(ctx, types.StringType, server.Environment.Addresses)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Addresses = serverAddresses
	state.ServerClustered = types.BoolValue(server.Environment.ServerClustered)
	state.ServerName = types.StringValue(server.Environment.ServerName)

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

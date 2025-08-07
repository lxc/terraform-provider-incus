package server

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type ServerDataSourceModel struct {
	IsClustered types.Bool   `tfsdk:"is_clustered"`
	Members     types.List   `tfsdk:"members"`
	Remote      types.String `tfsdk:"remote"`
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
			"is_clustered": schema.BoolAttribute{
				Computed: true,
			},

			"members": schema.ListAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"addresses": types.ListType{
							ElemType: types.StringType,
						},
						"server_name": types.StringType,
						"status":      types.StringType,
					},
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func getMemberObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"addresses": types.ListType{
				ElemType: types.StringType,
			},
			"server_name": types.StringType,
			"status":      types.StringType,
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

type serverEnvironment struct {
	addresses  []string
	serverName string
	status     string
}

func (d *ServerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ServerDataSourceModel

	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()

	serverProvider, err := d.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// get server environment of the default target to determine, if we are talking to a cluster.
	server, _, err := serverProvider.GetServer()
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve server environment", err.Error())
		return
	}

	state.IsClustered = types.BoolValue(server.Environment.ServerClustered)

	// short circuit if server is not clustered
	if !server.Environment.ServerClustered {
		members, diags := toMembersListType(ctx, []serverEnvironment{
			{
				addresses:  server.Environment.Addresses,
				serverName: server.Environment.ServerName,
				status:     "Online", // status of the server is not returned by the API, but since we just communicated with the server, we considere it "Online".
			},
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		state.Members = members

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
		return
	}

	// get cluster members
	clusterMembers, err := serverProvider.GetClusterMembers()
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve cluster members", err.Error())
		return
	}

	serverEnvs := make([]serverEnvironment, 0, len(clusterMembers))

	for _, clusterMember := range clusterMembers {
		// get server environment for each cluster member
		server, _, err := serverProvider.UseTarget(clusterMember.ServerName).GetServer()
		if err != nil {
			resp.Diagnostics.AddError("Failed to retrieve cluster member environment", err.Error())
			return
		}

		serverEnvs = append(serverEnvs, serverEnvironment{
			addresses:  server.Environment.Addresses,
			serverName: server.Environment.ServerName,
			status:     clusterMember.Status,
		})
	}

	members, diags := toMembersListType(ctx, serverEnvs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Members = members

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func toMembersListType(ctx context.Context, serverEnvs []serverEnvironment) (types.List, diag.Diagnostics) {
	memberObjectType := getMemberObjectType()
	nilList := types.ListNull(memberObjectType)

	memberList := make([]attr.Value, 0, len(serverEnvs))
	for _, serverEnv := range serverEnvs {
		memberAddresses, diags := types.ListValueFrom(ctx, types.StringType, serverEnv.addresses)
		if diags.HasError() {
			return nilList, diags
		}

		memberMap := map[string]attr.Value{
			"addresses":   memberAddresses,
			"server_name": types.StringValue(serverEnv.serverName),
			"status":      types.StringValue(serverEnv.status),
		}

		memberObject, diags := types.ObjectValue(memberObjectType.AttrTypes, memberMap)
		if diags.HasError() {
			return nilList, diags
		}

		memberList = append(memberList, memberObject)
	}

	return types.ListValue(memberObjectType, memberList)
}

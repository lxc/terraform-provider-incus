package cluster

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

type ClusterDataSourceModel struct {
	IsClustered types.Bool   `tfsdk:"is_clustered"`
	Members     types.List   `tfsdk:"members"`
	Remote      types.String `tfsdk:"remote"`
}

type ClusterDataSource struct {
	provider *provider_config.IncusProviderConfig
}

func NewClusterDataSource() datasource.DataSource {
	return &ClusterDataSource{}
}

func (d *ClusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_cluster", req.ProviderTypeName)
}

func (d *ClusterDataSource) Schema(_ context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"is_clustered": schema.BoolAttribute{
				Computed: true,
			},

			"members": schema.ListAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
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
			"server_name": types.StringType,
			"status":      types.StringType,
		},
	}
}

func (d *ClusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type clusterMemberItem struct {
	serverName string
	status     string
}

func (d *ClusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ClusterDataSourceModel

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

	// get cluster environment
	cluster, _, err := serverProvider.GetCluster()
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve cluster environment", err.Error())
		return
	}

	state.IsClustered = types.BoolValue(cluster.Enabled)

	// fall back if not clustered
	if !cluster.Enabled {
		members, diags := toMembersListType(ctx, []clusterMemberItem{
			{
				serverName: "",
				status:     "Online",
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

	clusterMemberItems := make([]clusterMemberItem, 0, len(clusterMembers))

	for _, clusterMember := range clusterMembers {
		clusterMemberItems = append(clusterMemberItems, clusterMemberItem{
			serverName: clusterMember.ServerName,
			status:     clusterMember.Status,
		})
	}

	members, diags := toMembersListType(ctx, clusterMemberItems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Members = members

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func toMembersListType(_ context.Context, clusterMemberItems []clusterMemberItem) (types.List, diag.Diagnostics) {
	memberObjectType := getMemberObjectType()
	nilList := types.ListNull(memberObjectType)

	memberList := make([]attr.Value, 0, len(clusterMemberItems))
	for _, clusterMember := range clusterMemberItems {
		memberMap := map[string]attr.Value{
			"server_name": types.StringValue(clusterMember.serverName),
			"status":      types.StringValue(clusterMember.status),
		}

		memberObject, diags := types.ObjectValue(memberObjectType.AttrTypes, memberMap)
		if diags.HasError() {
			return nilList, diags
		}

		memberList = append(memberList, memberObject)
	}

	return types.ListValue(memberObjectType, memberList)
}

package clustering

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
	Members     types.Map    `tfsdk:"members"`
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

			"members": schema.MapAttribute{
				Computed: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"address":        types.StringType,
						"architecture":   types.StringType,
						"description":    types.StringType,
						"failure_domain": types.StringType,
						"groups": types.ListType{
							ElemType: types.StringType,
						},
						"roles": types.ListType{
							ElemType: types.StringType,
						},
						"status": types.StringType,
					},
				},
			},

			"remote": schema.StringAttribute{
				Optional: true,
			},
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
	address       string
	architecture  string
	description   string
	failureDomain string
	groups        []string
	name          string
	roles         []string
	status        string
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

	if !cluster.Enabled {
		memberObjectType := getMemberObjectType()
		nilMap := types.MapNull(memberObjectType)

		state.Members = nilMap

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
			address:       clusterMember.URL,
			architecture:  clusterMember.Architecture,
			description:   clusterMember.Description,
			failureDomain: clusterMember.FailureDomain,
			groups:        clusterMember.Groups,
			name:          clusterMember.ServerName,
			roles:         clusterMember.Roles,
			status:        clusterMember.Status,
		})
	}

	members, diags := toMembersMapType(ctx, clusterMemberItems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Members = members

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func toMembersMapType(ctx context.Context, clusterMemberItems []clusterMemberItem) (types.Map, diag.Diagnostics) {
	memberObjectType := getMemberObjectType()
	nilMap := types.MapNull(memberObjectType)

	memberMap := make(map[string]attr.Value, len(clusterMemberItems))
	for _, clusterMember := range clusterMemberItems {
		groups, diags := types.ListValueFrom(ctx, types.StringType, clusterMember.groups)
		if diags.HasError() {
			return nilMap, diags
		}

		roles, diags := types.ListValueFrom(ctx, types.StringType, clusterMember.roles)
		if diags.HasError() {
			return nilMap, diags
		}

		memberObjectMap := map[string]attr.Value{
			"address":        types.StringValue(clusterMember.address),
			"architecture":   types.StringValue(clusterMember.architecture),
			"failure_domain": types.StringValue(clusterMember.failureDomain),
			"description":    types.StringValue(clusterMember.description),
			"roles":          roles,
			"groups":         groups,
			"status":         types.StringValue(clusterMember.status),
		}

		memberObject, diags := types.ObjectValue(memberObjectType.AttrTypes, memberObjectMap)
		if diags.HasError() {
			return nilMap, diags
		}

		memberMap[clusterMember.name] = memberObject
	}

	return types.MapValue(memberObjectType, memberMap)
}

func getMemberObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"address":        types.StringType,
			"architecture":   types.StringType,
			"description":    types.StringType,
			"failure_domain": types.StringType,
			"groups": types.ListType{
				ElemType: types.StringType,
			},
			"roles": types.ListType{
				ElemType: types.StringType,
			},
			"status": types.StringType,
		},
	}
}

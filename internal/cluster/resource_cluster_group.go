package cluster

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"

	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type ClusterGroupModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
	Members     types.Set    `tfsdk:"members"`
}

type ClusterGroupResource struct {
	provider *provider_config.IncusProviderConfig
}

func NewClusterGroupResource() resource.Resource {
	return &ClusterGroupResource{}
}

func (r *ClusterGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_cluster_group", req.ProviderTypeName)
}

func (r *ClusterGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"remote": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},

			"members": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					// Prevent empty values.
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
		},
	}
}

func (r *ClusterGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	data := req.ProviderData
	if data == nil {
		return
	}

	provider, ok := data.(*provider_config.IncusProviderConfig)
	if !ok {
		resp.Diagnostics.Append(errors.NewProviderDataTypeError(req.ProviderData))
		return
	}

	r.provider = provider
}

func (r *ClusterGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ClusterGroupModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	members, diags := ToMemberSet(ctx, plan.Members)
	diags.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	clusterGroup := api.ClusterGroupsPost{
		Name: plan.Name.ValueString(),
		ClusterGroupPut: api.ClusterGroupPut{
			Description: plan.Description.ValueString(),
			Config:      config,
			Members:     members,
		},
	}

	err = server.CreateClusterGroup(clusterGroup)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create cluster group %q", clusterGroup.Name), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ClusterGroupModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ClusterGroupModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	clusterGroupName := plan.Name.ValueString()
	_, etag, err := server.GetClusterGroup(clusterGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster group %q", clusterGroupName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	members, diags := ToMemberSet(ctx, plan.Members)
	diags.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedClusterGroup := api.ClusterGroupPut{
		Description: plan.Description.ValueString(),
		Config:      config,
		Members:     members,
	}

	err = server.UpdateClusterGroup(clusterGroupName, updatedClusterGroup, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update cluster group %q", clusterGroupName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ClusterGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ClusterGroupModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	server, err := r.provider.InstanceServer(remote, "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	clusterGroupName := state.Name.ValueString()
	clusterGroup, etag, err := server.GetClusterGroup(clusterGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cluster group %q", clusterGroupName), err.Error())
		return
	}

	// Before we can delete a cluster group, we need to remove all members first.
	updatedClusterGroup := api.ClusterGroupPut{
		Description: clusterGroup.Description,
	}

	err = server.UpdateClusterGroup(clusterGroupName, updatedClusterGroup, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove members from cluster group %q", clusterGroupName), err.Error())
		return
	}

	err = server.DeleteClusterGroup(clusterGroupName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cluster group %q", clusterGroupName), err.Error())
	}
}

func (r *ClusterGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "cluster_group",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

// SyncState fetches the server's current state for a cluster group and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r *ClusterGroupResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m ClusterGroupModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	clusterGroupName := m.Name.ValueString()
	clusterGroup, _, err := server.GetClusterGroup(clusterGroupName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve cluster group %q", clusterGroupName), err.Error())
		return respDiags
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(clusterGroup.Config), m.Config)
	respDiags.Append(diags...)

	members, diags := ToMemberSetType(ctx, clusterGroup.Members)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return respDiags
	}

	m.Name = types.StringValue(clusterGroup.Name)
	m.Description = types.StringValue(clusterGroup.Description)
	m.Config = config
	m.Members = members

	return tfState.Set(ctx, &m)
}

// ToMemberSet converts members of type types.Set into []string.
func ToMemberSet(ctx context.Context, memberList types.Set) ([]string, diag.Diagnostics) {
	members := make([]string, 0, len(memberList.Elements()))
	diags := memberList.ElementsAs(ctx, &members, false)

	return members, diags
}

// ToMemberSetType converts []string into members of type types.Set.
func ToMemberSetType(ctx context.Context, members []string) (types.Set, diag.Diagnostics) {
	nilList := types.SetNull(types.StringType)

	if len(members) == 0 {
		return nilList, nil
	}

	return types.SetValueFrom(ctx, types.StringType, members)
}

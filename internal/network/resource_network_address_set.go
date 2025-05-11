package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

// NetworkAddressSetModel resource data model that matches the schema.
type NetworkAddressSetModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Addresses   types.List   `tfsdk:"addresses"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Config      types.Map    `tfsdk:"config"`
}

// NetworkAddressSet represent Incus network address set resource.
type NetworkAddressSet struct {
	provider *provider_config.IncusProviderConfig
}

func NewNetworkAddressSet() resource.Resource {
	return &NetworkAddressSet{}
}

func (r *NetworkAddressSet) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_address_set", req.ProviderTypeName)
}

func (r *NetworkAddressSet) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"addresses": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"project": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
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
		},
	}
}

func (r *NetworkAddressSet) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NetworkAddressSet) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkAddressSetModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	addresses := make([]string, 0, len(plan.Addresses.Elements()))
	diags = plan.Addresses.ElementsAs(ctx, &addresses, false)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	networkAddressSetName := plan.Name.ValueString()
	networkAddressSetDescription := plan.Description.ValueString()

	networkAddressSetReq := api.NetworkAddressSetsPost{
		NetworkAddressSetPut: api.NetworkAddressSetPut{
			Addresses:   addresses,
			Description: networkAddressSetDescription,
			Config:      config,
		},
		NetworkAddressSetPost: api.NetworkAddressSetPost{
			Name: networkAddressSetName,
		},
	}

	err = server.CreateNetworkAddressSet(networkAddressSetReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network address set %q", networkAddressSetName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkAddressSet) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkAddressSetModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkAddressSet) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkAddressSetModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkAddressSetName := plan.Name.ValueString()
	_, etag, err := server.GetNetworkAddressSet(networkAddressSetName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network address set %q", networkAddressSetName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	addresses := make([]string, 0, len(plan.Addresses.Elements()))
	diags = plan.Addresses.ElementsAs(ctx, &addresses, false)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	networkAddressSetReq := api.NetworkAddressSetPut{
		Addresses:   addresses,
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	err = server.UpdateNetworkAddressSet(networkAddressSetName, networkAddressSetReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network address set %q", networkAddressSetName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *NetworkAddressSet) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkAddressSetModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	networkAddressSetName := state.Name.ValueString()
	err = server.DeleteNetworkAddressSet(networkAddressSetName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network address set %q", networkAddressSetName), err.Error())
	}
}

func (r *NetworkAddressSet) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "network_address_set",
		RequiredFields: []string{"name"},
	}

	fields, diags := meta.ParseImportID(req.ID)
	if diags != nil {
		resp.Diagnostics.Append(diags)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}
}

func (r *NetworkAddressSet) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m NetworkAddressSetModel) diag.Diagnostics {
	networkAddressSetName := m.Name.ValueString()
	networkAddressSet, _, err := server.GetNetworkAddressSet(networkAddressSetName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		return diag.Diagnostics{diag.NewErrorDiagnostic(
			fmt.Sprintf("Failed to retrieve network address set %q", networkAddressSetName), err.Error(),
		)}
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(networkAddressSet.Config), m.Config)
	if diags.HasError() {
		return diags
	}

	addresses, diags := types.ListValueFrom(ctx, types.StringType, networkAddressSet.Addresses)
	if diags.HasError() {
		return diags
	}

	m.Name = types.StringValue(networkAddressSet.Name)
	m.Description = types.StringValue(networkAddressSet.Description)
	m.Addresses = addresses
	m.Config = config

	return tfState.Set(ctx, &m)
}

package server

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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

// ServerModel resource data model that matches the schema.
type ServerModel struct {
	Remote types.String `tfsdk:"remote"`
	Config types.Map    `tfsdk:"config"`
	Target types.String `tfsdk:"target"`
}

// ServerResource represent Incus server resource.
type ServerResource struct {
	provider *provider_config.IncusProviderConfig
}

func NewServerResource() resource.Resource {
	return &ServerResource{}
}

func (r *ServerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_server", req.ProviderTypeName)
}

func (r *ServerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
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

			"target": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *ServerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ServerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.provider.ServerResourceMux.Lock()
	defer r.provider.ServerResourceMux.Unlock()

	var plan ServerModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	target := plan.Target.ValueString()
	serverProvider, err := r.provider.InstanceServer(remote, "", target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	server, eTag, err := serverProvider.GetServer()
	if err != nil {
		resp.Diagnostics.AddError("Failed to get server details", err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, managedKeys := compileConfig(server.Config, nil, userConfig)

	updatedServer := api.ServerPut{
		Config: config,
	}

	err = serverProvider.UpdateServer(updatedServer, eTag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update server config", err.Error())
		return
	}

	diags = r.syncState(ctx, &resp.State, serverProvider, plan, managedKeys)
	resp.Diagnostics.Append(diags...)
}

func (r *ServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.provider.ServerResourceMux.Lock()
	defer r.provider.ServerResourceMux.Unlock()

	var state ServerModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	target := state.Target.ValueString()
	serverProvider, err := r.provider.InstanceServer(remote, "", target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	userState, diags := common.ToConfigMap(ctx, state.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	managedKeys := make([]string, 0, len(userState))
	for k := range userState {
		managedKeys = append(managedKeys, k)
	}

	diags = r.syncState(ctx, &resp.State, serverProvider, state, managedKeys)
	resp.Diagnostics.Append(diags...)
}

func (r *ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.provider.ServerResourceMux.Lock()
	defer r.provider.ServerResourceMux.Unlock()

	var plan ServerModel
	var state ServerModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	target := plan.Target.ValueString()
	serverProvider, err := r.provider.InstanceServer(remote, "", target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	server, eTag, err := serverProvider.GetServer()
	if err != nil {
		resp.Diagnostics.AddError("Failed to get server config", err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	userState, diags := common.ToConfigMap(ctx, state.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, managedKeys := compileConfig(server.Config, userState, userConfig)

	updatedServer := api.ServerPut{
		Config: config,
	}

	err = serverProvider.UpdateServer(updatedServer, eTag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update server config", err.Error())
		return
	}

	diags = r.syncState(ctx, &resp.State, serverProvider, plan, managedKeys)
	resp.Diagnostics.Append(diags...)
}

func (r *ServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	r.provider.ServerResourceMux.Lock()
	defer r.provider.ServerResourceMux.Unlock()

	var state ServerModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	target := state.Target.ValueString()
	serverProvider, err := r.provider.InstanceServer(remote, "", target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	server, eTag, err := serverProvider.GetServer()
	if err != nil {
		resp.Diagnostics.AddError("Failed to get server config", err.Error())
		return
	}

	userState, diags := common.ToConfigMap(ctx, state.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, managedKeys := compileConfig(server.Config, userState, nil)

	updatedServer := api.ServerPut{
		Config: config,
	}

	err = serverProvider.UpdateServer(updatedServer, eTag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update server config", err.Error())
		return
	}

	diags = r.syncState(ctx, &resp.State, serverProvider, state, managedKeys)
	resp.Diagnostics.Append(diags...)
}

// syncState fetches the server's current state for the server configuration and
// updates the provided model. It then applies this updated model as the new
// state in Terraform.
func (r *ServerResource) syncState(ctx context.Context, tfState *tfsdk.State, serverProvider incus.InstanceServer, m ServerModel, managedKeys []string) diag.Diagnostics {
	var respDiags diag.Diagnostics

	server, _, err := serverProvider.GetServer()
	if err != nil {
		respDiags.AddError("Failed to retrieve server config", err.Error())
		return respDiags
	}

	stateConfig := make(map[string]*string, len(managedKeys))

	for k, v := range server.Config {
		if !slices.Contains(managedKeys, k) {
			continue
		}

		stateConfig[k] = &v
	}

	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	if diags.HasError() {
		return diags
	}

	m.Config = config

	return tfState.Set(ctx, &m)
}

// compileConfig generates the desired target state of the config and returns
// the list of config keys, which are managed by this resource, which are all
// keys present in the plan.
//
// The desired target state is formed by:
//
//   - Any key the currently exists in the config (pre dating Terraform or managed by a different resource).
//   - Adding respectively overwritting all keys from with the values from the plan.
//   - Setting all keys to empty string, that are in the state but no longer in the plan.
func compileConfig(resourceConfig, stateConfig, planConfig map[string]string) (map[string]string, []string) {
	managedKeys := make([]string, 0, len(planConfig))
	for k := range planConfig {
		managedKeys = append(managedKeys, k)
	}

	targetResourceConfig := make(map[string]string, len(resourceConfig)+len(stateConfig)+len(planConfig))
	maps.Copy(targetResourceConfig, resourceConfig)
	maps.Copy(targetResourceConfig, planConfig)

	for k := range stateConfig {
		if slices.Contains(managedKeys, k) {
			continue
		}

		// Set value to empty string ("") for Incus to remove the key.
		targetResourceConfig[k] = ""
	}

	return targetResourceConfig, managedKeys
}

package server

import (
	"context"
	"encoding/json"
	"fmt"
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

const privateStatePreExistingConfigKeys = "pre-existing-config-keys"

// ServerModel resource data model that matches the schema.
type ServerModel struct {
	Remote types.String `tfsdk:"remote"`
	Config types.Map    `tfsdk:"config"`
	Target types.String `tfsdk:"target"`
}

// ComputedKeys returns list of computed config keys.
func (ServerModel) ComputedKeys() []string {
	return []string{}
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

	preExistingKeys := make([]string, 0, len(server.Config))
	for k := range server.Config {
		_, ok := userConfig[k]
		if !ok {
			preExistingKeys = append(preExistingKeys, k)
		}
	}

	config := common.MergeConfig(server.Config, userConfig, preExistingKeys)
	updatedServer := api.ServerPut{
		Config: config,
	}

	err = serverProvider.UpdateServer(updatedServer, eTag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update server config", err.Error())
		return
	}

	preExistingKeysJSON, err := json.Marshal(preExistingKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal pre existing keys", err.Error())
		return
	}
	resp.Private.SetKey(ctx, privateStatePreExistingConfigKeys, preExistingKeysJSON)

	diags = r.SyncState(ctx, &resp.State, serverProvider, plan, preExistingKeys)
	resp.Diagnostics.Append(diags...)
}

func (r *ServerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	preExistingKeysJSON, diags := req.Private.GetKey(ctx, privateStatePreExistingConfigKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var preExistingKeys []string
	err = json.Unmarshal(preExistingKeysJSON, &preExistingKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal pre existing keys", err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, serverProvider, state, preExistingKeys)
	resp.Diagnostics.Append(diags...)
}

func (r *ServerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
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
		resp.Diagnostics.AddError("Failed to get server config", err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	preExistingKeysJSON, diags := req.Private.GetKey(ctx, privateStatePreExistingConfigKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var preExistingKeys []string
	err = json.Unmarshal(preExistingKeysJSON, &preExistingKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal pre existing keys", err.Error())
		return
	}

	remainingPreExistingKeys := make([]string, 0, len(preExistingKeys))
	for _, k := range preExistingKeys {
		_, ok := userConfig[k]
		if !ok {
			remainingPreExistingKeys = append(remainingPreExistingKeys, k)
		}
	}

	config := common.MergeConfig(server.Config, userConfig, remainingPreExistingKeys)
	updatedServer := api.ServerPut{
		Config: config,
	}

	err = serverProvider.UpdateServer(updatedServer, eTag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update server config", err.Error())
		return
	}

	remainingPreExistingKeysJSON, err := json.Marshal(remainingPreExistingKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to marshal remaining pre existing keys", err.Error())
		return
	}
	resp.Private.SetKey(ctx, privateStatePreExistingConfigKeys, remainingPreExistingKeysJSON)

	diags = r.SyncState(ctx, &resp.State, serverProvider, plan, remainingPreExistingKeys)
	resp.Diagnostics.Append(diags...)
}

func (r *ServerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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

	preExistingKeysJSON, diags := req.Private.GetKey(ctx, privateStatePreExistingConfigKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var preExistingKeys []string
	err = json.Unmarshal(preExistingKeysJSON, &preExistingKeys)
	if err != nil {
		resp.Diagnostics.AddError("Failed to unmarshal pre existing keys", err.Error())
		return
	}

	config := make(map[string]string, len(server.Config))
	for k, v := range server.Config {
		if slices.Contains(preExistingKeys, k) {
			config[k] = v
		}
	}

	updatedServer := api.ServerPut{
		Config: config,
	}

	err = serverProvider.UpdateServer(updatedServer, eTag)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update server config", err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, serverProvider, state, preExistingKeys)
	resp.Diagnostics.Append(diags...)
}

// SyncState fetches the server's current state for the server configuration and
// updates the provided model. It then applies this updated model as the new
// state in Terraform.
func (r *ServerResource) SyncState(ctx context.Context, tfState *tfsdk.State, serverProvider incus.InstanceServer, m ServerModel, preExistingKeys []string) diag.Diagnostics {
	var respDiags diag.Diagnostics

	server, _, err := serverProvider.GetServer()
	if err != nil {
		respDiags.AddError("Failed to retrieve server config", err.Error())
		return respDiags
	}

	// Extract user defined config and merge it with current config state.
	stateConfig := common.StripConfig(server.Config, m.Config, preExistingKeys)

	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	if diags.HasError() {
		return diags
	}

	m.Config = config

	return tfState.Set(ctx, &m)
}

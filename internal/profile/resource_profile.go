package profile

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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

type ProfileModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Devices     types.Set    `tfsdk:"device"`
	Config      types.Map    `tfsdk:"config"`
}

// ProfileResource represent Incus profile resource.
type ProfileResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewProfileResource returns a new profile resource.
func NewProfileResource() resource.Resource {
	return &ProfileResource{}
}

// Metadata for profile resource.
func (r ProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_profile", req.ProviderTypeName)
}

// Schema for profile resource.
func (r ProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

		Blocks: map[string]schema.Block{
			"device": schema.SetNestedBlock{
				Description: "Profile device",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Device name",
						},

						"type": schema.StringAttribute{
							Required:    true,
							Description: "Device type",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"none", "disk", "nic", "unix-char",
									"unix-block", "usb", "gpu", "infiniband",
									"proxy", "unix-hotplug", "tpm", "pci",
								),
							},
						},

						"properties": schema.MapAttribute{
							Required:    true,
							Description: "Device properties",
							ElementType: types.StringType,
							Validators: []validator.Map{
								// Prevent empty values.
								mapvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProfileResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	priorState := req.State
	plannedState := req.Plan

	// Detection creation: no prior state
	if priorState.Raw.IsNull() {
		var profileName types.String
		diags := plannedState.GetAttribute(ctx, path.Root("name"), &profileName)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// We only print the warning here and take care of the correct changes in the ProfileResource#Create
		if profileName.ValueString() == "default" {
			resp.Diagnostics.AddWarning(
				"Default Profile Creation Attempted",
				"'default' profile cannot be created in an Incus project. Instead, its config and devices will be updated accordingly.",
			)
		}
		return
	}

	// Detect deletion: no planned state
	if plannedState.Raw.IsNull() {
		var profileName types.String
		diags := priorState.GetAttribute(ctx, path.Root("name"), &profileName)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		// We only print the warning here and take care of the correct changes in the ProfileResource#Delete
		if profileName.ValueString() == "default" {
			resp.Diagnostics.AddWarning(
				"Default Profile Deletion Attempted",
				"'default' profile cannot be deleted in an Incus project. Instead, its config and devices will be set to the Incus default profile state.",
			)
		}
		return
	}
}

func (r ProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProfileModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Name.ValueString() == "default" {
		r.importAndUpdateDefaultProfile(ctx, plan, resp)
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Convert profile config and devices to map.
	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	profile := api.ProfilesPost{
		Name: plan.Name.ValueString(),
		ProfilePut: api.ProfilePut{
			Description: plan.Description.ValueString(),
			Config:      config,
			Devices:     devices,
		},
	}

	err = server.CreateProfile(profile)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create profile %q", profile.Name), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProfileModel

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

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProfileModel

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

	profileName := plan.Name.ValueString()
	_, etag, err := server.GetProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing profile %q", profileName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Update profile.
	profile := api.ProfilePut{
		Description: plan.Description.ValueString(),
		Config:      config,
		Devices:     devices,
	}

	err = server.UpdateProfile(profileName, profile, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update profile %q", profileName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProfileModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.Name.ValueString() == "default" {
		r.resetDefaultProfile(ctx, state, resp)
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	profileName := state.Name.ValueString()
	err = server.DeleteProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove profile %q", profileName), err.Error())
	}
}

func (r ProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "profile",
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

// SyncState fetches the server's current state for a profile and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r ProfileResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m ProfileModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	profileName := m.Name.ValueString()
	profile, _, err := server.GetProfile(profileName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve profile %q", profileName), err.Error())
		return respDiags
	}

	// Convert config state and devices into schema types.
	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(profile.Config), m.Config)
	respDiags.Append(diags...)

	devices, diags := common.ToDeviceSetType(ctx, profile.Devices)
	respDiags.Append(diags...)

	m.Name = types.StringValue(profile.Name)
	m.Description = types.StringValue(profile.Description)
	m.Devices = devices
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

func (r ProfileResource) importAndUpdateDefaultProfile(ctx context.Context, plan ProfileModel, resp *resource.CreateResponse) {
	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	profileName := plan.Name.ValueString()

	// 1. We import the default profile.
	var id string
	if project != "" {
		id = fmt.Sprintf("%s/%s", project, profileName)
	} else {
		id = profileName
	}

	meta := common.ImportMetadata{
		ResourceName:   "profile",
		RequiredFields: []string{"name"},
	}

	fields, diag := meta.ParseImportID(id)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}

	// 2. After we have imported the default profile, we can now update it.
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	_, etag, err := server.GetProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing profile %q", profileName), err.Error())
		return
	}

	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	profile := api.ProfilePut{
		Description: plan.Description.ValueString(),
		Config:      config,
		Devices:     devices,
	}

	err = server.UpdateProfile(profileName, profile, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update profile %q", profileName), err.Error())
		return
	}

	// 3. Sync Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ProfileResource) resetDefaultProfile(ctx context.Context, plan ProfileModel, resp *resource.DeleteResponse) {
	remote := plan.Remote.ValueString()
	projectName := plan.Project.ValueString()
	profileName := plan.Name.ValueString()

	server, err := r.provider.InstanceServer(remote, projectName, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	_, etag, err := server.GetProfile(profileName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing profile %q", profileName), err.Error())
		return
	}

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	defaultDevices := make(map[string]map[string]string)
	defaultConfig := make(map[string]string)

	// We must keep the root device if features.profiles is set to false.
	project, _, err := server.GetProject(projectName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve project %q", projectName), err.Error())
		return
	}

	if project.Config["features.profiles"] == "false" {
		if value, exists := devices["root"]; exists {
			defaultDevices["root"] = value
		}
	}

	profile := api.ProfilePut{
		Description: plan.Description.ValueString(),
		Config:      defaultConfig,
		Devices:     defaultDevices,
	}

	err = server.UpdateProfile(profileName, profile, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update profile %q", profileName), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

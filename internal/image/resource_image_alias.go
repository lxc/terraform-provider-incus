package image

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

// ImageAlias resource data model that matches the schema.
type ImageAliasModel struct {
	Alias       types.String `tfsdk:"alias"`
	Description types.String `tfsdk:"description"`
	Project     types.String `tfsdk:"project"`
	Remote      types.String `tfsdk:"remote"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

// ImageAliasResource represent Incus image alias resource.
type ImageAliasResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewImageAliasResource return a new image alias resource.
func NewImageAliasResource() resource.Resource {
	return &ImageAliasResource{}
}

func (r ImageAliasResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_image_alias", req.ProviderTypeName)
}

func (r ImageAliasResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"alias": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"description": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
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

			"fingerprint": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *ImageAliasResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ImageAliasResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ImageAliasModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Alias.IsNull() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"The alias must be set.",
		)

		return
	}
}

func (r ImageAliasResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ImageAliasModel

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

	alias := plan.Alias.ValueString()
	aliasDescription := plan.Description.ValueString()
	imageFingerprint := plan.Fingerprint.ValueString()

	imageAliasReq := api.ImageAliasesPost{}
	imageAliasReq.Name = alias
	imageAliasReq.Description = aliasDescription
	imageAliasReq.Target = imageFingerprint

	if err := server.CreateImageAlias(imageAliasReq); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create alias %q for cache image with fingerprint %q", alias, imageFingerprint), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ImageAliasResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ImageAliasModel

	// Fetch resource model from Terraform state.
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
	diags = r.SyncState(ctx, &req.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r ImageAliasResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ImageAliasModel
	var state ImageAliasModel

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
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	imageServer, err := r.provider.ImageServer(remote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	oldAlias := state.Alias.ValueString()
	imageAlias, _, err := imageServer.GetImageAlias(oldAlias)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cached image with alias %q", oldAlias), err.Error())
		return
	}

	imageFingerprint := imageAlias.Target
	newAlias := plan.Alias.ValueString()
	imageAliasReq := api.ImageAliasesEntryPost{
		Name: newAlias,
	}

	if err = server.RenameImageAlias(oldAlias, imageAliasReq); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to rename alias %q for cache image with fingerprint %q", oldAlias, imageFingerprint), err.Error())
		return
	}

	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ImageAliasResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan ImageAliasModel

	diags := req.State.Get(ctx, &plan)
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

	imageServer, err := r.provider.ImageServer(remote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	alias := plan.Alias.ValueString()
	imageAlias, _, err := imageServer.GetImageAlias(alias)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve cached image with alias %q", alias), err.Error())
		return
	}

	imageFingerprint := imageAlias.Target

	if err = server.DeleteImageAlias(alias); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete alias %q for cached image with %q", alias, imageFingerprint), err.Error())
		return
	}
}

// SyncState fetches the server's current state for an image alias and
// updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r ImageAliasResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m ImageAliasModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	alias := m.Alias.ValueString()

	imageAlias, _, err := server.GetImageAlias(alias)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}
		respDiags.AddError(fmt.Sprintf("Failed to retrieve cached image with alias %q", alias), err.Error())
		return respDiags
	}

	m.Alias = types.StringValue(imageAlias.Name)
	m.Description = types.StringValue(imageAlias.Description)

	if m.Fingerprint.IsNull() {
		m.Fingerprint = types.StringNull()
	} else {
		m.Fingerprint = types.StringValue(imageAlias.Target)
	}

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

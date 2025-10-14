package network

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// NetworkPeerModel resource data model that matches the schema.
type NetworkPeerModel struct {
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	Config            types.Map    `tfsdk:"config"`
	Project           types.String `tfsdk:"project"`
	Network           types.String `tfsdk:"network"`
	TargetProject     types.String `tfsdk:"target_project"`
	TargetNetwork     types.String `tfsdk:"target_network"`
	Remote            types.String `tfsdk:"remote"`
	Type              types.String `tfsdk:"type"`
	TargetIntegration types.String `tfsdk:"target_integration"`
	Status            types.String `tfsdk:"status"`
}

// IncusNetworkPeerResource represent Incus network peer resource.
type IncusNetworkPeerResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewNetworkPeerResource returns a new network peer resource.
func NewNetworkPeerResource() resource.Resource {
	return &IncusNetworkPeerResource{}
}

func (r IncusNetworkPeerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_network_peer", req.ProviderTypeName)
}

func (r IncusNetworkPeerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},

			"target_project": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"network": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"target_network": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"target_integration": schema.StringAttribute{
				Optional: true,
				Computed: true,
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

			"status": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *IncusNetworkPeerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r IncusNetworkPeerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan NetworkPeerModel

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

	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)

	if resp.Diagnostics.HasError() {
		return
	}

	peerName := plan.Name.ValueString()
	description := plan.Description.ValueString()
	networkName := plan.Network.ValueString()
	targetProject := plan.TargetProject.ValueString()
	targetNetwork := plan.TargetNetwork.ValueString()
	_type := plan.Type.ValueString()
	targetIntegration := plan.TargetIntegration.ValueString()

	networkPeerReq := api.NetworkPeersPost{
		Name:              peerName,
		TargetProject:     targetProject,
		TargetNetwork:     targetNetwork,
		Type:              _type,
		TargetIntegration: targetIntegration,
		NetworkPeerPut: api.NetworkPeerPut{
			Description: description,
			Config:      config,
		},
	}

	// Create Network Peer.
	err = server.CreateNetworkPeer(networkName, networkPeerReq)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create network peer %q", peerName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r IncusNetworkPeerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NetworkPeerModel

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

func (r IncusNetworkPeerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan NetworkPeerModel

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

	config, diag := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diag...)

	if resp.Diagnostics.HasError() {
		return
	}

	peerName := plan.Name.ValueString()
	networkName := plan.Network.ValueString()
	description := plan.Description.ValueString()

	peerReq := api.NetworkPeerPut{
		Description: description,
		Config:      config,
	}

	// Update network peer.
	_, etag, err := server.GetNetworkPeer(networkName, peerName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing network peer %q", peerName), err.Error())
		return
	}

	err = server.UpdateNetworkPeer(networkName, peerName, peerReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update network peer %q", peerName), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r IncusNetworkPeerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NetworkPeerModel

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

	peerName := state.Name.ValueString()
	networkName := state.Network.ValueString()

	err = server.DeleteNetworkPeer(networkName, peerName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove network peer %q", peerName), err.Error())
	}
}

// SyncState fetches the server's current state for an network peer
// and updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r IncusNetworkPeerResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m NetworkPeerModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	peerName := m.Name.ValueString()
	networkName := m.Network.ValueString()

	networkPeer, _, err := server.GetNetworkPeer(networkName, peerName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve network peer %q", peerName), err.Error())
		return respDiags
	}

	config, diags := common.ToConfigMapType(ctx, common.ToNullableConfig(networkPeer.Config), m.Config)
	respDiags.Append(diags...)

	m.Description = types.StringValue(networkPeer.Description)
	m.TargetNetwork = types.StringValue(networkPeer.TargetNetwork)
	m.TargetProject = types.StringValue(networkPeer.TargetProject)
	m.Type = types.StringValue(networkPeer.Type)
	m.TargetIntegration = types.StringValue(networkPeer.TargetIntegration)
	m.Status = types.StringValue(networkPeer.Status)
	m.Config = config

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

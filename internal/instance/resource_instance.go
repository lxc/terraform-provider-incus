package instance

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
	"github.com/mitchellh/go-homedir"

	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
	"github.com/lxc/terraform-provider-incus/internal/utils"
)

type InstanceModel struct {
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	Image          types.String `tfsdk:"image"`
	Ephemeral      types.Bool   `tfsdk:"ephemeral"`
	Running        types.Bool   `tfsdk:"running"`
	WaitForConfigs types.Set    `tfsdk:"wait_for"`
	Profiles       types.List   `tfsdk:"profiles"`
	Devices        types.Set    `tfsdk:"device"`
	Files          types.Set    `tfsdk:"file"`
	Config         types.Map    `tfsdk:"config"`
	Project        types.String `tfsdk:"project"`
	Remote         types.String `tfsdk:"remote"`
	Target         types.String `tfsdk:"target"`
	SourceInstance types.Object `tfsdk:"source_instance"`
	SourceFile     types.String `tfsdk:"source_file"`

	// Computed.
	IPv4   types.String `tfsdk:"ipv4_address"`
	IPv6   types.String `tfsdk:"ipv6_address"`
	MAC    types.String `tfsdk:"mac_address"`
	Status types.String `tfsdk:"status"`
}

func (m InstanceModel) IsContainer() bool {
	return m.Type.ValueString() == "container"
}

func (m InstanceModel) IsVirtualMachine() bool {
	return m.Type.ValueString() == "virtual-machine"
}

type SourceInstanceModel struct {
	Project  types.String `tfsdk:"project"`
	Name     types.String `tfsdk:"name"`
	Snapshot types.String `tfsdk:"snapshot"`
}

type WaitForModel struct {
	Type  types.String `tfsdk:"type"`
	Delay types.String `tfsdk:"delay"`
	Nic   types.String `tfsdk:"nic"`
}

func (m WaitForModel) IsAgent() bool {
	return m.Type.ValueString() == "agent"
}

func (m WaitForModel) IsDelay() bool {
	return m.Type.ValueString() == "delay"
}

func (m WaitForModel) IsIPv4() bool {
	return m.Type.ValueString() == "ipv4"
}

func (m WaitForModel) IsIPv6() bool {
	return m.Type.ValueString() == "ipv6"
}

func (m WaitForModel) IsReady() bool {
	return m.Type.ValueString() == "ready"
}

// InstanceResource represent Incus instance resource.
type InstanceResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewInstanceResource returns a new instance resource.
func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

func (r InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_instance", req.ProviderTypeName)
}

func (r InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("container"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("container", "virtual-machine"),
				},
			},

			"image": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.Expressions{
							path.MatchRoot("source_instance"),
							path.MatchRoot("source_file"),
						}...,
					),
				},
			},

			"ephemeral": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"running": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},

			// If profiles are null, use "default" profile.
			// If profiles length is 0, no profiles are applied.
			"profiles": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					// Prevent empty values.
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

			"target": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},

			"config": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Validators: []validator.Map{
					mapvalidator.KeysAre(configKeyValidator{}),
				},
			},

			"source_instance": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"project": schema.StringAttribute{
						Required: true,
					},
					"name": schema.StringAttribute{
						Required: true,
					},
					"snapshot": schema.StringAttribute{
						Optional: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.Expressions{
							path.MatchRoot("source_file"),
						}...,
					),
				},
			},

			"source_file": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(
						path.Expressions{
							path.MatchRoot("description"),
							path.MatchRoot("type"),
							path.MatchRoot("ephemeral"),
							path.MatchRoot("profiles"),
							path.MatchRoot("file"),
							path.MatchRoot("config"),
						}...,
					),
				},
			},

			// Computed.

			"ipv4_address": schema.StringAttribute{
				Computed: true,
			},

			"ipv6_address": schema.StringAttribute{
				Computed: true,
			},

			"mac_address": schema.StringAttribute{
				Computed: true,
			},

			"status": schema.StringAttribute{
				Computed: true,
			},
		},

		Blocks: map[string]schema.Block{
			"wait_for": schema.SetNestedBlock{
				Description: "Wait for instance to be ready",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("agent", "delay", "ipv4", "ipv6", "ready"),
							},
						},
						"delay": schema.StringAttribute{
							Optional: true,
						},
						"nic": schema.StringAttribute{
							Optional: true,
						},
					},
				},
			},

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

			"file": schema.SetNestedBlock{
				Description: "Upload file to instance",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"content": schema.StringAttribute{
							Optional: true,
						},

						"source_path": schema.StringAttribute{
							Optional: true,
						},

						"target_path": schema.StringAttribute{
							Required: true,
						},

						"uid": schema.Int64Attribute{
							Optional: true,
						},

						"gid": schema.Int64Attribute{
							Optional: true,
						},

						"mode": schema.StringAttribute{
							Optional: true,
							Computed: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},

						"create_directories": schema.BoolAttribute{
							Optional: true,
						},

						// Append is here just to satisfy the IncusFile model.
						"append": schema.BoolAttribute{
							Computed: true,
							Default:  booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func (r *InstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InstanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// If resource is being destroyed req.Config will be null.
	// In such case there is no need for plan modification.
	if req.Config.Raw.IsNull() {
		return
	}

	var profiles types.List
	req.Config.GetAttribute(ctx, path.Root("profiles"), &profiles)

	// If profiles are null, set "default" profile.
	if profiles.IsNull() {
		resp.Plan.SetAttribute(ctx, path.Root("profiles"), []string{"default"})
	}
}

func (r InstanceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if req.Config.Raw.IsNull() {
		return
	}

	var config InstanceModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	running := true
	ephemeral := false

	if !config.Ephemeral.IsNull() && !config.Ephemeral.IsUnknown() {
		ephemeral = config.Ephemeral.ValueBool()
	}

	if !config.Running.IsNull() && !config.Running.IsUnknown() {
		running = config.Running.ValueBool()
	}

	// Ephemeral instance cannot be stopped.
	if ephemeral && !running {
		resp.Diagnostics.AddAttributeError(
			path.Root("running"),
			fmt.Sprintf("Instance %q is ephemeral and cannot be stopped", config.Name.ValueString()),
			fmt.Sprintf("Ephemeral instances are removed when stopped, therefore attribute %q must be set to %q.", "running", "true"),
		)
	}

	if !config.SourceFile.IsNull() {
		// With `incus import`, a storage pool can be provided optionally.
		// In order to support the same behavior with source_file,
		// a single device entry of type `disk` is allowed with exactly two properties
		// `path` and `pool` being set. For `path`, the only accepted value is `/`.
		if len(config.Devices.Elements()) > 0 {
			if len(config.Devices.Elements()) > 1 {
				resp.Diagnostics.AddError(
					"Invalid Configuration",
					"Only one device entry is supported with source_file.",
				)
				return
			}

			deviceList := make([]common.DeviceModel, 0, 1)
			diags = config.Devices.ElementsAs(ctx, &deviceList, false)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			if len(deviceList[0].Properties.Elements()) != 2 {
				resp.Diagnostics.AddError(
					"Invalid Configuration",
					`Exactly two device properties named "path" and "pool" need to be provided with source_file.`,
				)
				return
			}

			properties := make(map[string]string, 2)
			diags = deviceList[0].Properties.ElementsAs(ctx, &properties, false)
			if diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}

			_, poolOK := properties["pool"]
			path, pathOK := properties["path"]

			if !poolOK || !pathOK || path != "/" {
				resp.Diagnostics.AddError(
					"Invalid Configuration",
					`Exactly two device properties named "path" and "pool" need to be provided with source_file. For "path", the only accepted value is "/".`,
				)
				return
			}
		}
	}

	if len(config.WaitForConfigs.Elements()) > 0 {
		validateWaitFor(ctx, config, resp)
	}

	if !config.Files.IsNull() {
		if !config.Running.IsNull() && !config.Running.ValueBool() {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				"Files can only be pushed to running instances.",
			)
		}

		if config.IsVirtualMachine() {
			validateWaitForAgentWithFiles(ctx, config, resp)
		}
	}
}

// validateWaitFor validates the wait_for configuration blocks.
func validateWaitFor(ctx context.Context, config InstanceModel, resp *resource.ValidateConfigResponse) {
	waitForList := make([]WaitForModel, 0, 1)
	diags := config.WaitForConfigs.ElementsAs(ctx, &waitForList, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	for _, waitFor := range waitForList {
		if waitFor.IsAgent() || waitFor.IsReady() {
			if !waitFor.Nic.IsNull() {
				resp.Diagnostics.AddError(
					"Invalid Configuration",
					`"nic" can only be set when type is set to "ipv4" or "ipv6.`,
				)
			}
		}

		if waitFor.IsAgent() && (config.Type.IsNull() || config.IsContainer()) {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`"type" can only be set to "delay","ipv4" or "ipv6" when type is set to "container".`,
			)
		}

		if config.IsVirtualMachine() && waitFor.IsReady() {
			resp.Diagnostics.AddError(
				"Invalid Configuration",
				`"type" can only be set to "agent","ipv4" or "ipv6" when type is set to "virtual-machine".`,
			)
		}

		if waitFor.IsDelay() {
			if !waitFor.Nic.IsNull() {
				resp.Diagnostics.AddError(
					"Invalid Configuration",
					`"nic" can only be set when type is set to "ipv4" or "ipv6.`,
				)
			}

			if waitFor.Delay.IsNull() {
				resp.Diagnostics.AddError(
					"Invalid Configuration",
					`"delay" is required when type is set to "delay".`,
				)
			}
		}
	}
}

// validateWaitForAgentWithFiles validates the wait_for configuration for the type agent.
func validateWaitForAgentWithFiles(ctx context.Context, config InstanceModel, resp *resource.ValidateConfigResponse) {
	waitForMap, diags := ToWaitForConfigMap(ctx, config.WaitForConfigs)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	_, found := waitForMap["agent"]
	if !found {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			`Wait for "agent" is required when files are uploaded to the instance.`,
		)
		return
	}
}

func (r InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	if !plan.Image.IsNull() {
		diags = r.createInstanceFromImage(ctx, server, plan)
		resp.Diagnostics.Append(diags...)
	} else if !plan.SourceFile.IsNull() {
		diags = r.createInstanceFromSourceFile(ctx, server, plan)
		resp.Diagnostics.Append(diags...)
	} else if !plan.SourceInstance.IsNull() {
		diags = r.createInstanceFromSourceInstance(ctx, server, plan)
		resp.Diagnostics.Append(diags...)
	} else {
		if plan.Running.ValueBool() {
			resp.Diagnostics.AddError("running must be set to false if the instance is created without image or source_instance", "")
			return
		}

		diags = r.createInstanceWithoutImage(ctx, server, plan)
		resp.Diagnostics.Append(diags...)
	}

	if diags.HasError() {
		return
	}

	instanceName := plan.Name.ValueString()

	// Partially update state, to make terraform aware of
	// an existing instance.
	diags = resp.State.SetAttribute(ctx, path.Root("name"), instanceName)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// We must ensure that the instance is running before we can upload files.
	if plan.Running.ValueBool() || (!plan.Files.IsNull() && !plan.Files.IsUnknown()) {
		diag := startInstance(ctx, server, instanceName)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}

		// Take the wait_for configurations into account.
		if len(plan.WaitForConfigs.Elements()) > 0 {
			diags := waitFor(ctx, server, instanceName, plan.WaitForConfigs)
			if diags != nil {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	// Upload files.
	if !plan.Files.IsNull() && !plan.Files.IsUnknown() {
		files, diags := common.ToFileMap(ctx, plan.Files)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, f := range files {
			err := common.InstanceFileUpload(server, instanceName, f)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to instance %q", instanceName), err.Error())
				return
			}
		}
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

// Update updates the instance in the following order:
// - Ensure instance state (stopped/running)
// - Update configuration (config, devices, profiles)
// - Upload files
// - Run exec commands
func (r InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan InstanceModel
	var state InstanceModel

	// Fetch resource model from Terraform plan.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := plan.Name.ValueString()
	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return
	}

	// First ensure the desired state of the instance (stopped/running).
	// This ensures we fail fast if instance runs into an issue.
	if plan.Running.ValueBool() && !isInstanceOperational(*instanceState) {
		diag := startInstance(ctx, server, instanceName)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}

		// If instance is freshly started, we also take the wait for configurations into account.
		if len(plan.WaitForConfigs.Elements()) > 0 {
			diags := waitFor(ctx, server, instanceName, plan.WaitForConfigs)
			if diags != nil {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	} else if !plan.Running.ValueBool() && !isInstanceStopped(*instanceState) {
		// Stop the instance gracefully.
		_, diag := stopInstance(ctx, server, instanceName, false)
		if diag != nil {
			resp.Diagnostics.Append(diag)
			return
		}
	}

	// Get instance.
	instance, etag, err := server.GetInstance(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing instance %q", instanceName), err.Error())
		return
	}

	// First extract profiles, devices, config and config state.
	// Then merge user defined config with instance config (state).
	profiles, diags := ToProfileList(ctx, plan.Profiles)
	resp.Diagnostics.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	resp.Diagnostics.Append(diags...)

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)

	config := common.MergeConfig(instance.Config, userConfig, plan.ComputedKeys())

	if resp.Diagnostics.HasError() {
		return
	}

	newInstance := api.InstancePut{
		Description:  plan.Description.ValueString(),
		Ephemeral:    plan.Ephemeral.ValueBool(),
		Architecture: instance.Architecture,
		Restore:      instance.Restore,
		Stateful:     instance.Stateful,
		Config:       config,
		Profiles:     profiles,
		Devices:      devices,
	}

	// Update the instance.
	opUpdate, err := server.UpdateInstance(instanceName, newInstance, etag)
	if err == nil {
		// Wait for the instance to be updated.
		err = opUpdate.Wait()
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update instance %q", instance.Name), err.Error())
		return
	}

	oldFiles, diags := common.ToFileMap(ctx, state.Files)
	resp.Diagnostics.Append(diags...)

	newFiles, diags := common.ToFileMap(ctx, plan.Files)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Remove files that are no longer present in newFiles.
	for k, f := range oldFiles {
		_, ok := newFiles[k]
		if ok {
			continue
		}

		targetPath := f.TargetPath.ValueString()
		err := common.InstanceFileDelete(server, instanceName, targetPath)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file from instance %q", instanceName), err.Error())
			return
		}
	}

	// Upload new files.
	for k, f := range newFiles {
		_, ok := oldFiles[k]
		if ok {
			continue
		}

		err := common.InstanceFileUpload(server, instanceName, f)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to instance %q", instanceName), err.Error())
			return
		}
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceModel

	// Fetch resource model from Terraform state.
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote := state.Remote.ValueString()
	project := state.Project.ValueString()
	target := state.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := state.Name.ValueString()

	// Force stop the instance, because we are deleting it anyway.
	isFound, diag := stopInstance(ctx, server, instanceName, true)
	if diag != nil {
		// Ephemeral instances will be removed when stopped.
		if !isFound {
			return
		}

		resp.Diagnostics.Append(diag)
		return
	}

	// Delete the instance.
	opDelete, err := server.DeleteInstance(instanceName)
	if err == nil {
		// Wait for the instance to be deleted.
		err = opDelete.WaitContext(ctx)
	}

	if err != nil && !errors.IsNotFoundError(err) {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove instance %q", instanceName), err.Error())
	}
}

func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "instance",
		RequiredFields: []string{"name"},
		AllowedOptions: []string{"image"},
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

// SyncState fetches the server's current state for an instance and updates
// the provided model. It then applies this updated model as the new state
// in Terraform.
func (r InstanceResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m InstanceModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	instanceName := m.Name.ValueString()
	instance, _, err := server.GetInstance(instanceName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve instance %q", instanceName), err.Error())
		return respDiags
	}

	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		respDiags.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return respDiags
	}

	// Reset IPv4, IPv6, and MAC addresses. If case instance has lost
	// network connectivity, we should reflect that in state.
	m.IPv4 = types.StringNull()
	m.IPv6 = types.StringNull()
	m.MAC = types.StringNull()

	// First there is an access_interface set, extract IPv4, IPv6, and
	// MAC addresses from it.
	accIface, ok := instance.Config["user.access_interface"]
	if ok {
		ipv4, ipv6, mac, _, found := getAddresses(accIface, instanceState.Network[accIface])

		if found {
			if ipv4 != "" {
				m.IPv4 = types.StringValue(ipv4)
			}

			if ipv6 != "" {
				m.IPv6 = types.StringValue(ipv6)
			}

			if mac != "" {
				m.MAC = types.StringValue(mac)
			}
		}
	} else {
		// If the above wasn't successful, try to automatically determine
		// the IPv4, IPv6, and MAC addresses.
		ipv4, ipv6, mac, _, found := findAddresses(instanceState)

		if found {
			if ipv4 != "" {
				m.IPv4 = types.StringValue(ipv4)
			}

			if ipv6 != "" {
				m.IPv6 = types.StringValue(ipv6)
			}

			if mac != "" {
				m.MAC = types.StringValue(mac)
			}
		}
	}

	// Extract user defined config and merge it with current resource config.
	stateConfig := common.StripConfig(instance.Config, m.Config, m.ComputedKeys())

	// Convert config, profiles, and devices into schema type.
	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	respDiags.Append(diags...)

	profiles, diags := ToProfileListType(ctx, instance.Profiles)
	respDiags.Append(diags...)

	devices, diags := common.ToDeviceSetType(ctx, instance.Devices)
	respDiags.Append(diags...)

	if respDiags.HasError() {
		return respDiags
	}

	if !m.SourceFile.IsNull() && !m.Devices.IsNull() {
		// Using device to signal the storage pool is a special case, which is not
		// reflected on the instance state and therefore we need to compensate here
		// in order to prevent inconsistent provider results.
		devices = m.Devices
	}

	m.Name = types.StringValue(instance.Name)
	m.Type = types.StringValue(instance.Type)
	m.Description = types.StringValue(instance.Description)
	m.Ephemeral = types.BoolValue(instance.Ephemeral)
	m.Status = types.StringValue(instance.Status)
	m.Profiles = profiles
	m.Devices = devices
	m.Config = config

	// Update "running" attribute based on the instance's current status.
	// This way, terraform will detect the change if the current status
	// does not match the expected one.
	m.Running = types.BoolValue(instanceState.Status == api.Running.String())

	m.Target = types.StringValue("")
	if server.IsClustered() || instance.Location != "none" {
		m.Target = types.StringValue(instance.Location)
	}

	return tfState.Set(ctx, &m)
}

func (r InstanceResource) createInstanceFromImage(ctx context.Context, server incus.InstanceServer, plan InstanceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	instance, diags := prepareInstancesPost(ctx, plan)
	if diags.HasError() {
		return diags
	}

	image := plan.Image.ValueString()
	imageRemote := ""
	imageParts := strings.SplitN(image, ":", 2)
	if len(imageParts) == 2 {
		imageRemote = imageParts[0]
		image = imageParts[1]
	}

	var imageServer incus.ImageServer
	if imageRemote == "" {
		imageServer = server
	} else {
		var err error
		imageServer, err = r.provider.ImageServer(imageRemote)
		if err != nil {
			diags.Append(errors.NewImageServerError(err))
			return diags
		}
	}

	var imageInfo *api.Image

	// Gather info about source image.
	conn, _ := imageServer.GetConnectionInfo()
	if conn.Protocol != "incus" {
		// Optimisation for public servers.
		imageInfo = &api.Image{}
		imageInfo.Public = true
		imageInfo.Fingerprint = image
		instance.Source.Alias = image
	} else {
		alias, _, err := imageServer.GetImageAlias(image)
		if err == nil {
			image = alias.Target
			instance.Source.Alias = image
		}

		imageInfo, _, err = imageServer.GetImage(image)
		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to retrieve image info for instance %q", instance.Name), err.Error())
			return diags
		}
	}

	opCreate, err := server.CreateInstanceFromImage(imageServer, *imageInfo, instance)
	// Initialize the instance. Instance will not be running after this call.
	if err == nil {
		// Wait for the instance to be created.
		err = opCreate.Wait()
	}

	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to create instance %q", instance.Name), err.Error())
		return diags
	}

	return diags
}

func (r InstanceResource) createInstanceFromSourceFile(ctx context.Context, server incus.InstanceServer, plan InstanceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	name := plan.Name.ValueString()

	var poolName string

	if len(plan.Devices.Elements()) > 0 {
		// Only one device is expected, this is ensured by ValidateConfig.
		deviceList := make([]common.DeviceModel, 0, 1)
		diags = plan.Devices.ElementsAs(ctx, &deviceList, false)
		if diags.HasError() {
			return diags
		}

		// Exactly two properties named "path" and "pool" are expected, this is ensured by ValidateConfig.
		properties := make(map[string]string, 2)
		diags = deviceList[0].Properties.ElementsAs(ctx, &properties, false)
		if diags.HasError() {
			return diags
		}

		poolName = properties["pool"]
	}

	srcFile := plan.SourceFile.ValueString()

	path, err := homedir.Expand(srcFile)
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to determine source_file: %q", srcFile), err.Error())
		return diags
	}

	file, err := os.Open(path)
	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to open source_file: %q", path), err.Error())
		return diags
	}

	defer func() { _ = file.Close() }()

	createArgs := incus.InstanceBackupArgs{
		BackupFile: file,
		PoolName:   poolName,
		Name:       name,
	}

	op, err := server.CreateInstanceFromBackup(createArgs)
	if err == nil {
		err = op.Wait()
	}

	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to create instance: %q", name), err.Error())
		return diags
	}

	return diags
}

func (r InstanceResource) createInstanceFromSourceInstance(ctx context.Context, destServer incus.InstanceServer, plan InstanceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var sourceInstanceModel SourceInstanceModel

	diags = plan.SourceInstance.As(ctx, &sourceInstanceModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return diags
	}

	name := plan.Name.ValueString()

	remote := plan.Remote.ValueString()
	sourceInstanceProject := sourceInstanceModel.Project.ValueString()
	target := plan.Target.ValueString()
	sourceServer, err := r.provider.InstanceServer(remote, sourceInstanceProject, target)
	if err != nil {
		diags.Append(errors.NewInstanceServerError(err))
		return diags
	}

	sourceInstanceName := sourceInstanceModel.Name.ValueString()

	if sourceInstanceModel.Snapshot.IsNull() {
		args := incus.InstanceCopyArgs{
			Name:              name,
			Live:              true,
			InstanceOnly:      true,
			Refresh:           false,
			AllowInconsistent: false,
		}

		sourceInstance, _, err := sourceServer.GetInstance(sourceInstanceName)
		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to retrieve instance %q", sourceInstanceName), err.Error())
			return diags
		}

		// Extract profiles, devices and config.
		profiles, diags := ToProfileList(ctx, plan.Profiles)
		diags.Append(diags...)

		devices, diags := common.ToDeviceMap(ctx, plan.Devices)
		diags.Append(diags...)

		config, diags := common.ToConfigMap(ctx, plan.Config)
		diags.Append(diags...)

		if diags.HasError() {
			return diags
		}

		sourceInstance.Profiles = profiles

		// Allow setting additional config keys
		for key, value := range config {
			sourceInstance.Config[key] = value
		}

		// Allow setting device overrides
		for k, m := range devices {
			if sourceInstance.Devices[k] == nil {
				sourceInstance.Devices[k] = m
				continue
			}

			for key, value := range m {
				sourceInstance.Devices[k][key] = value
			}
		}

		for k := range sourceInstance.Config {
			if !instanceIncludeWhenCopying(k, true) {
				delete(sourceInstance.Config, k)
			}
		}

		opCreate, err := destServer.CopyInstance(sourceServer, *sourceInstance, &args)
		if err == nil {
			err = opCreate.Wait()
		}

		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to create instance %q", name), err.Error())
			return diags
		}

		return diags
	} else {
		args := incus.InstanceSnapshotCopyArgs{
			Name: name,
			Live: true,
		}

		sourceSnapshotName := sourceInstanceModel.Snapshot.ValueString()
		sourceSnapshot, _, err := sourceServer.GetInstanceSnapshot(sourceInstanceName, sourceSnapshotName)
		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to retrieve snapshot %q from instance %q", sourceSnapshotName, sourceInstanceName), err.Error())
			return diags
		}

		// Extract profiles, devices and config.
		profiles, diags := ToProfileList(ctx, plan.Profiles)
		diags.Append(diags...)

		devices, diags := common.ToDeviceMap(ctx, plan.Devices)
		diags.Append(diags...)

		config, diags := common.ToConfigMap(ctx, plan.Config)
		diags.Append(diags...)

		if diags.HasError() {
			return diags
		}

		sourceSnapshot.Profiles = profiles

		// Allow setting additional config keys
		for key, value := range config {
			sourceSnapshot.Config[key] = value
		}

		// Allow setting device overrides
		for k, m := range devices {
			if sourceSnapshot.Devices[k] == nil {
				sourceSnapshot.Devices[k] = m
				continue
			}

			for key, value := range m {
				sourceSnapshot.Devices[k][key] = value
			}
		}

		for k := range sourceSnapshot.Config {
			if !instanceIncludeWhenCopying(k, true) {
				delete(sourceSnapshot.Config, k)
			}
		}

		opCreate, err := destServer.CopyInstanceSnapshot(sourceServer, sourceInstanceName, *sourceSnapshot, &args)
		if err == nil {
			err = opCreate.Wait()
		}

		if err != nil {
			diags.AddError(fmt.Sprintf("Failed to create instance %q from snapshot %q", name, sourceSnapshotName), err.Error())
			return diags
		}

		return diags
	}
}

func instanceIncludeWhenCopying(configKey string, remoteCopy bool) bool {
	if configKey == "volatile.base_image" {
		return true // Include volatile.base_image always as it can help optimize copies.
	}

	if configKey == "volatile.last_state.idmap" && !remoteCopy {
		return true // Include volatile.last_state.idmap when doing local copy to avoid needless remapping.
	}

	if strings.HasPrefix(configKey, "volatile.") {
		return false // Exclude all other volatile keys.
	}

	return true // Keep all other keys.
}

func (r InstanceResource) createInstanceWithoutImage(ctx context.Context, server incus.InstanceServer, plan InstanceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	instance, diags := prepareInstancesPost(ctx, plan)
	if diags.HasError() {
		return diags
	}

	instance.Source = api.InstanceSource{
		Type: "none",
	}

	opCreate, err := server.CreateInstance(instance)
	// Initialize the instance. Instance will not be running after this call.
	if err == nil {
		// Wait for the instance to be created.
		err = opCreate.Wait()
	}

	if err != nil {
		diags.AddError(fmt.Sprintf("Failed to create instance %q", instance.Name), err.Error())
		return diags
	}

	return diags
}

func prepareInstancesPost(ctx context.Context, plan InstanceModel) (api.InstancesPost, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Extract profiles, devices and config.
	profiles, diags := ToProfileList(ctx, plan.Profiles)
	diags.Append(diags...)

	devices, diags := common.ToDeviceMap(ctx, plan.Devices)
	diags.Append(diags...)

	config, diags := common.ToConfigMap(ctx, plan.Config)
	diags.Append(diags...)

	if diags.HasError() {
		return api.InstancesPost{}, diags
	}

	instance := api.InstancesPost{
		Name: plan.Name.ValueString(),
		Type: api.InstanceType(plan.Type.ValueString()),
		InstancePut: api.InstancePut{
			Description: plan.Description.ValueString(),
			Ephemeral:   plan.Ephemeral.ValueBool(),
			Config:      config,
			Profiles:    profiles,
			Devices:     devices,
		},
	}
	return instance, nil
}

// ComputedKeys returns list of computed config keys.
func (_ InstanceModel) ComputedKeys() []string {
	return []string{
		"environment.",
		"image.",
		"volatile.",
	}
}

// ToProfileList converts profiles of type types.List into []string.
//
// If profiles are null, use "default" profile.
// If profiles lengeth is 0, no profiles are applied.
func ToProfileList(ctx context.Context, profileList types.List) ([]string, diag.Diagnostics) {
	if profileList.IsNull() {
		return []string{"default"}, nil
	}

	profiles := make([]string, 0, len(profileList.Elements()))
	diags := profileList.ElementsAs(ctx, &profiles, false)

	return profiles, diags
}

// ToProfileListType converts []string into profiles of type types.List.
func ToProfileListType(ctx context.Context, profiles []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, profiles)
}

// startInstance starts an instance with the given name. It also waits
// for it to become fully operational.
func startInstance(ctx context.Context, server incus.InstanceServer, instanceName string) diag.Diagnostic {
	st, etag, err := server.GetInstanceState(instanceName)
	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
	}

	// Return if the instance is "Running" or "Ready".
	if isInstanceRunning(*st) {
		return nil
	}

	startReq := api.InstanceStatePut{
		Action:  "start",
		Force:   false,
		Timeout: utils.ContextTimeout(ctx, 3*time.Minute),
	}

	// Start the instance.
	op, err := server.UpdateInstanceState(instanceName, startReq, etag)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to start instance %q", instanceName), err.Error())
	}

	instanceStartedCheck := func() (any, string, error) {
		st, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return st, "Error", err
		}

		return st, st.Status, nil
	}

	// Even though op.Wait has completed, wait until we can see
	// the instance is fully started via a new API call.
	_, err = waitForState(ctx, instanceStartedCheck, api.Running.String())
	if err != nil {
		return diag.NewErrorDiagnostic(fmt.Sprintf("Failed to wait for instance %q to start", instanceName), err.Error())
	}

	return nil
}

// stopInstance stops an instance with the given name. It waits for its
// status to become Stopped or the instance to be removed (not found) in
// case of an ephemeral instance. In the latter case, false is returned
// along an error.
func stopInstance(ctx context.Context, server incus.InstanceServer, instanceName string, force bool) (bool, diag.Diagnostic) {
	st, etag, err := server.GetInstanceState(instanceName)
	if err != nil {
		return true, diag.NewErrorDiagnostic(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
	}

	// Return if the instance is already stopped.
	if isInstanceStopped(*st) {
		return true, nil
	}

	stopReq := api.InstanceStatePut{
		Action:  "stop",
		Force:   force,
		Timeout: utils.ContextTimeout(ctx, 3*time.Minute),
	}

	// Stop the instance.
	op, err := server.UpdateInstanceState(instanceName, stopReq, etag)
	if err == nil {
		err = op.WaitContext(ctx)
	}

	if err != nil {
		return true, diag.NewErrorDiagnostic(fmt.Sprintf("Failed to stop instance %q", instanceName), err.Error())
	}

	instanceStoppedCheck := func() (any, string, error) {
		st, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return st, "Error", err
		}

		return st, st.Status, nil
	}

	// Even though op.Wait has completed, wait until we can see
	// the instance is stopped via a new API call.
	_, err = waitForState(ctx, instanceStoppedCheck, api.Stopped.String())
	if err != nil {
		found := !errors.IsNotFoundError(err)
		return found, diag.NewErrorDiagnostic(fmt.Sprintf("Failed to wait for instance %q to stop", instanceName), err.Error())
	}

	return true, nil
}

// waitFor waits for the instance with the given name to reach the desired
// state. It returns an error if the instance does not reach the desired
// state within the given timeout.
func waitFor(ctx context.Context, server incus.InstanceServer, instanceName string, waitFor types.Set) diag.Diagnostics {
	var diags diag.Diagnostics

	waitForMap, diags := ToWaitForConfigMap(ctx, waitFor)
	if diags.HasError() {
		return diags
	}

	for waitForModelType, waitForModel := range waitForMap {
		switch waitForModelType {
		case "agent":
			diags = waitForInstanceAgent(ctx, server, instanceName)
		case "delay":
			duration := waitForModel.Delay.ValueString()
			diags = waitForInstanceWithDelay(ctx, server, instanceName, duration)
		case "ipv4", "ipv6":
			nic := waitForModel.Nic.ValueString()
			diags = waitForInstanceNetwork(ctx, server, instanceName, waitForModelType, nic)
		case "ready":
			diags = waitForInstanceToBeReady(ctx, server, instanceName)
		default:
			diags.AddError(fmt.Sprintf("Invalid value for wait_for: %q", waitForModelType), "")
		}
	}

	return diags
}

// waitForInstanceAgent waits for an instance with the given name to have
// the Incus agent fully operational.
func waitForInstanceAgent(ctx context.Context, server incus.InstanceServer, instanceName string) diag.Diagnostics {
	instanceAgentCheck := func() (any, string, error) {
		state, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return state, "Error", err
		}

		if isInstanceOperational(*state) {
			return state, "OK", nil
		}

		return state, "Waiting for instance to be ready", nil
	}

	_, err := waitForState(ctx, instanceAgentCheck, "OK")
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(fmt.Sprintf("Failed to wait for instance %q agent to be ready", instanceName), err.Error())
		return diags
	}

	return nil
}

// waitForInstanceWithDelay waits for an instance with the given name to wait
// for a given duration before continuing.
func waitForInstanceWithDelay(ctx context.Context, server incus.InstanceServer, instanceName string, delay string) diag.Diagnostics {
	instanceDelayCheck := func() (any, string, error) {
		state, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return state, "Error", err
		}

		duration, err := time.ParseDuration(delay)
		if err != nil {
			return state, "Error", err
		}

		time.Sleep(duration)

		return state, "OK", nil
	}

	_, err := waitForState(ctx, instanceDelayCheck, "OK")
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(fmt.Sprintf("Failed to wait for instance %q", instanceName), err.Error())
		return diags
	}

	return nil
}

// waitForInstanceNetwork waits for an instance with the given name to receive
// an IPv4 address on any interface (excluding loopback). This should be
// called only if the instance is running.
func waitForInstanceNetwork(ctx context.Context, server incus.InstanceServer, instanceName string, ipFamily string, nic string) diag.Diagnostics {
	instanceNetworkCheck := func() (any, string, error) {
		state, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return state, "Error", err
		}

		for iface, net := range state.Network {
			if iface == "lo" {
				continue
			}

			for _, ip := range net.Addresses {
				if ipFamily == "ipv4" && ip.Family == "inet" && (nic == "" || iface == nic) {
					return state, "OK", nil
				}
				if ipFamily == "ipv6" && ip.Family == "inet6" && (nic == "" || iface == nic) {
					return state, "OK", nil
				}
			}
		}

		return state, "Waiting for network", nil
	}

	_, err := waitForState(ctx, instanceNetworkCheck, "OK")
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(fmt.Sprintf("Failed to wait for instance %q to get an IP address", instanceName), err.Error())
		return diags
	}

	return nil
}

// waitForInstanceToBeReady waits for an instance with the given name to be
// ready.
func waitForInstanceToBeReady(ctx context.Context, server incus.InstanceServer, instanceName string) diag.Diagnostics {
	instanceReadyCheck := func() (any, string, error) {
		state, _, err := server.GetInstanceState(instanceName)
		if err != nil {
			return state, "Error", err
		}

		if isInstanceReady(*state) {
			return state, "OK", nil
		}

		return state, "Waiting for instance to be ready", nil
	}

	_, err := waitForState(ctx, instanceReadyCheck, "OK")
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(fmt.Sprintf("Failed to wait for instance %q to be ready", instanceName), err.Error())
		return diags
	}

	return nil
}

// waitForState waits until the provided function reports one of the target
// states. It returns either the resulting state or an error.
func waitForState(ctx context.Context, refreshFunc retry.StateRefreshFunc, targets ...string) (any, error) {
	stateRefreshConf := &retry.StateChangeConf{
		Refresh:    refreshFunc,
		Target:     targets,
		Timeout:    3 * time.Minute,
		MinTimeout: 2 * time.Second, // Timeout increases: 2, 4, 8, 10, 10, ...
		Delay:      2 * time.Second, // Delay before the first check/refresh.
	}

	return stateRefreshConf.WaitForStateContext(ctx)
}

// isInstanceOperational determines if an instance is fully operational based
// on its state. It returns true if the instance is running and the reported
// process count is positive. Checking for a positive process count is essential
// for virtual machines, which can report this metric only if the Incus agent has
// started and has established a connection to the Incus server.
func isInstanceOperational(s api.InstanceState) bool {
	return isInstanceRunning(s) && s.Processes > 0
}

// isInstanceRunning returns true if its status is either "Running" or "Ready".
func isInstanceRunning(s api.InstanceState) bool {
	return s.StatusCode == api.Running || s.StatusCode == api.Ready
}

// isInstanceRunning returns true if its status is "Ready".
func isInstanceReady(s api.InstanceState) bool {
	return s.StatusCode == api.Ready
}

// isInstanceStopped returns true if instance's status "Stopped".
func isInstanceStopped(s api.InstanceState) bool {
	return s.StatusCode == api.Stopped
}

// iface is a wrapper to store a map[string]api.InstanceStateNetwork as a slice.
type iface struct {
	api.InstanceStateNetwork
	Name string
}

// sortedInterfaces is used to sort a []iface from least to most desirable.
type sortedInterfaces []iface

func (s sortedInterfaces) Len() int {
	return len(s)
}

func (s sortedInterfaces) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortedInterfaces) Less(i, j int) bool {
	favorWithValue := func(a, b string) bool {
		if a == "" && b == "" {
			return false
		}

		if len(a) > 0 && len(b) > 0 {
			return false
		}
		return true
	}

	// Favor those with a host interface name.
	if favorWithValue(s[i].HostName, s[j].HostName) {
		return s[i].HostName == ""
	}

	// Favor those with a MAC address.
	if favorWithValue(s[i].Hwaddr, s[j].Hwaddr) {
		return s[i].Hwaddr == ""
	}

	// Favor those with addresses.
	hasIP := func(entry iface, family string) bool {
		for _, address := range entry.Addresses {
			if address.Scope != "global" || address.Family != family {
				continue
			}

			return true
		}

		return false
	}

	if hasIP(s[i], "inet") != hasIP(s[j], "inet") {
		return !hasIP(s[i], "inet")
	}

	if hasIP(s[i], "inet6") != hasIP(s[j], "inet6") {
		return !hasIP(s[i], "inet6")
	}

	return false
}

// findAddresses looks for the most optimal interface on the instance to return
// the IPv4, IPv6 and MAC address and interface name from.
func findAddresses(state *api.InstanceState) (string, string, string, string, bool) {
	if len(state.Network) == 0 {
		return "", "", "", "", false
	}

	ifaces := make(sortedInterfaces, 0, len(state.Network))
	for ifaceName, entry := range state.Network {
		ifaces = append(ifaces, iface{InstanceStateNetwork: entry, Name: ifaceName})
	}

	sort.Sort(sort.Reverse(ifaces))

	if ifaces[0].Name == "lo" {
		return "", "", "", "", false
	}

	return getAddresses(ifaces[0].Name, ifaces[0].InstanceStateNetwork)
}

// getAddresses returns the IPv4, IPv6 and MAC addresses for the interface.
func getAddresses(name string, entry api.InstanceStateNetwork) (string, string, string, string, bool) {
	var ipv4 string
	var ipv6 string

	for _, address := range entry.Addresses {
		if address.Scope != "global" {
			continue
		}

		if ipv4 == "" && address.Family == "inet" {
			ipv4 = address.Address
		}

		if ipv6 == "" && address.Family == "inet6" {
			ipv6 = address.Address
		}

		if ipv4 != "" && ipv6 != "" {
			break
		}
	}

	return ipv4, ipv6, entry.Hwaddr, name, true
}

// ToWaitForConfigMap converts wait_for from types.Set into map[string]WaitForModel.
func ToWaitForConfigMap(ctx context.Context, waitForSet types.Set) (map[string]WaitForModel, diag.Diagnostics) {
	if waitForSet.IsNull() || waitForSet.IsUnknown() {
		return make(map[string]WaitForModel), nil
	}

	waitForConfigs := make([]WaitForModel, 0, len(waitForSet.Elements()))
	diags := waitForSet.ElementsAs(ctx, &waitForConfigs, false)
	if diags.HasError() {
		return nil, diags
	}

	waitForMap := make(map[string]WaitForModel, len(waitForConfigs))
	for _, f := range waitForConfigs {
		waitForMap[f.Type.ValueString()] = f
	}

	return waitForMap, diags
}

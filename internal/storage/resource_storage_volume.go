package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"

	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
)

type StorageVolumeModel struct {
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Pool         types.String `tfsdk:"pool"`
	Type         types.String `tfsdk:"type"`
	ContentType  types.String `tfsdk:"content_type"`
	Project      types.String `tfsdk:"project"`
	Target       types.String `tfsdk:"target"`
	Remote       types.String `tfsdk:"remote"`
	Config       types.Map    `tfsdk:"config"`
	SourceVolume types.Object `tfsdk:"source_volume"`
	SourceFile   types.String `tfsdk:"source_file"`
	Files        types.Set    `tfsdk:"file"`

	// Computed.
	Location types.String `tfsdk:"location"`
}

// StorageVolumeResource represent Incus storage volume resource.
type StorageVolumeResource struct {
	provider *provider_config.IncusProviderConfig
}

type SourceVolumeModel struct {
	Pool   types.String `tfsdk:"pool"`
	Name   types.String `tfsdk:"name"`
	Remote types.String `tfsdk:"remote"`
}

// NewStorageVolumeResource returns a new storage volume resource.
func NewStorageVolumeResource() resource.Resource {
	return &StorageVolumeResource{}
}

func (r StorageVolumeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_storage_volume", req.ProviderTypeName)
}

func (r StorageVolumeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				PlanModifiers: []planmodifier.String{
					common.SetDefaultStringIfAllUndefined(
						types.StringValue(""),
						path.MatchRoot("source_volume"),
						path.MatchRoot("source_file"),
					),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("source_file"), path.MatchRoot("source_volume")),
				},
			},

			"pool": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("custom"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("source_file"), path.MatchRoot("source_volume")),
				},
			},

			"content_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					common.SetDefaultStringIfAllUndefined(
						types.StringValue("filesystem"),
						path.MatchRoot("source_volume"),
						path.MatchRoot("source_file"),
					),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("filesystem", "block"),
					stringvalidator.ConflictsWith(path.MatchRoot("source_file"), path.MatchRoot("source_volume")),
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
				PlanModifiers: []planmodifier.Map{
					common.SetDefaultMapIfAllUndefined(
						types.MapValueMust(types.StringType, map[string]attr.Value{}),
						path.MatchRoot("source_volume"),
						path.MatchRoot("source_file"),
					),
				},
				Validators: []validator.Map{
					mapvalidator.ConflictsWith(path.MatchRoot("source_file"), path.MatchRoot("source_volume")),
				},
			},

			"source_volume": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"pool": schema.StringAttribute{
						Required: true,
					},
					"name": schema.StringAttribute{
						Required: true,
					},
					"remote": schema.StringAttribute{
						Optional: true,
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(path.MatchRoot("source_file")),
				},
			},

			"source_file": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.ConflictsWith(path.MatchRoot("source_volume")),
				},
			},

			// Computed.

			"location": schema.StringAttribute{
				Computed: true,
			},
		},

		Blocks: map[string]schema.Block{
			"file": schema.SetNestedBlock{
				Description: "Upload file to storage volume",
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

func (r *StorageVolumeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r StorageVolumeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan StorageVolumeModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.SourceVolume.IsNull() {
		r.copyStoragePoolVolume(ctx, resp, &plan)
		return
	}

	if !plan.SourceFile.IsNull() {
		r.importStoragePoolVolume(ctx, resp, &plan)
		return
	}

	r.createStoragePoolVolume(ctx, resp, &plan)
}

func (r StorageVolumeResource) createStoragePoolVolume(ctx context.Context, resp *resource.CreateResponse, plan *StorageVolumeModel) {
	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Convert volume config to map.
	config, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolName := plan.Pool.ValueString()
	volName := plan.Name.ValueString()

	vol := api.StorageVolumesPost{
		Name:        plan.Name.ValueString(),
		Type:        plan.Type.ValueString(),
		ContentType: plan.ContentType.ValueString(),
		StorageVolumePut: api.StorageVolumePut{
			Description: plan.Description.ValueString(),
			Config:      config,
		},
	}

	err = server.CreateStoragePoolVolume(poolName, vol)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage volume %q", volName), err.Error())
		return
	}

	r.uploadFilesOnStoragePoolVolume(ctx, resp, plan)

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, *plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) copyStoragePoolVolume(ctx context.Context, resp *resource.CreateResponse, plan *StorageVolumeModel) {
	var sourceVolumeModel SourceVolumeModel

	diags := plan.SourceVolume.As(ctx, &sourceVolumeModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	dstProject := plan.Project.ValueString()
	dstTarget := plan.Target.ValueString()
	dstServer, err := r.provider.InstanceServer(plan.Remote.ValueString(), dstProject, dstTarget)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	srcServer, err := r.provider.InstanceServer(sourceVolumeModel.Remote.ValueString(), "", "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	dstName := plan.Name.ValueString()
	dstPool := plan.Pool.ValueString()
	srcName := sourceVolumeModel.Name.ValueString()
	srcPool := sourceVolumeModel.Pool.ValueString()

	dstVolID := fmt.Sprintf("%s/%s", dstPool, dstName)
	srcVolID := fmt.Sprintf("%s/%s", srcPool, srcName)

	srcVol := api.StorageVolume{
		Name: srcName,
		Type: "custom",
	}

	args := incus.StoragePoolVolumeCopyArgs{
		Name:       dstName,
		VolumeOnly: true,
	}

	opCopy, err := dstServer.CopyStoragePoolVolume(dstPool, srcServer, srcPool, srcVol, &args)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy storage volume %q -> %q", srcVolID, dstVolID), err.Error())
		return
	}

	err = opCopy.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy storage volume %q -> %q", srcVolID, dstVolID), err.Error())
		return
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, dstServer, *plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) uploadFilesOnStoragePoolVolume(ctx context.Context, resp *resource.CreateResponse, plan *StorageVolumeModel) {
	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	// Upload files.
	if !plan.Files.IsNull() && !plan.Files.IsUnknown() {
		files, diags := common.ToFileMap(ctx, plan.Files)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		poolName := plan.Pool.ValueString()
		volumeType := plan.Type.ValueString()
		volumeName := plan.Name.ValueString()

		for _, f := range files {
			err := common.VolumeFileUpload(server, poolName, volumeType, volumeName, f)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to volume %q in pool %q", volumeName, poolName), err.Error())
				return
			}
		}
	}
}

func (r StorageVolumeResource) importStoragePoolVolume(ctx context.Context, resp *resource.CreateResponse, plan *StorageVolumeModel) {
	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	target := plan.Target.ValueString()
	sourceFile := plan.SourceFile.ValueString()
	server, err := r.provider.InstanceServer(remote, project, target)
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	poolName := plan.Pool.ValueString()
	volName := plan.Name.ValueString()

	file, err := os.Open(sourceFile)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to open source file %q", sourceFile), err.Error())
		return
	}
	defer file.Close()

	args := incus.StorageVolumeBackupArgs{
		Name:       volName,
		BackupFile: file,
	}

	var opImport incus.Operation

	if strings.HasSuffix(file.Name(), ".iso") {
		opImport, err = server.CreateStoragePoolVolumeFromISO(poolName, args)
	} else {
		opImport, err = server.CreateStoragePoolVolumeFromBackup(poolName, args)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage volume from file %q", volName), err.Error())
		return
	}

	err = opImport.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create storage volume from file %q", volName), err.Error())
		return
	}

	// Update Terraform state.
	diags := r.SyncState(ctx, &resp.State, server, *plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state StorageVolumeModel

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

func (r StorageVolumeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan StorageVolumeModel
	var state StorageVolumeModel

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

	poolName := plan.Pool.ValueString()
	volName := plan.Name.ValueString()
	volType := plan.Type.ValueString()
	vol, etag, err := server.GetStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve existing storage volume %q", volName), err.Error())
		return
	}

	userConfig, diags := common.ToConfigMap(ctx, plan.Config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Merge volume config and user defined config.
	config := common.MergeConfig(vol.Config, userConfig, plan.ComputedKeys())

	volReq := api.StorageVolumePut{
		Description: plan.Description.ValueString(),
		Config:      config,
	}

	// Update volume.
	err = server.UpdateStoragePoolVolume(poolName, volType, volName, volReq, etag)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to update storage volume %q", volName), err.Error())
		return
	}

	targetResource := fmt.Sprintf("Volume %s/%s", poolName, volName)

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
		err := common.VolumeFileDelete(server, poolName, volType, volName, targetPath)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file from volume %q", targetResource), err.Error())
			return
		}
	}

	// Upload new files or update existing files if content has changed.
	for k, newFile := range newFiles {
		oldFile, exists := oldFiles[k]

		if !exists {
			err := common.VolumeFileUpload(server, poolName, volType, volName, newFile)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to volume %q", targetResource), err.Error())
				return
			}
			continue
		}

		contentChanged := common.HasFileContentChanged(newFile, oldFile)
		permissionsChanged := common.HasFilePermissionChanged(newFile, oldFile)

		if contentChanged || permissionsChanged {
			// Delete the old file first otherwise mode and ownership changes
			// will not be applied.
			targetPath := newFile.TargetPath.ValueString()
			err := common.VolumeFileDelete(server, poolName, volType, volName, targetPath)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file from volume %q", targetResource), err.Error())
				return
			}

			err = common.VolumeFileUpload(server, poolName, volType, volName, newFile)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload updated file to volume %q", targetResource), err.Error())
				return
			}
		}
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r StorageVolumeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state StorageVolumeModel

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

	poolName := state.Pool.ValueString()
	volName := state.Name.ValueString()
	volType := state.Type.ValueString()
	err = server.DeleteStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove storage pool %q", poolName), err.Error())
	}
}

func (r StorageVolumeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	meta := common.ImportMetadata{
		ResourceName:   "volume",
		RequiredFields: []string{"pool", "name"},
	}

	fields, diag := meta.ParseImportID(req.ID)
	if diag != nil {
		resp.Diagnostics.Append(diag)
		return
	}

	for k, v := range fields {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(k), v)...)
	}

	// Currently, only "custom" volumes can be imported.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), "custom")...)
}

// SyncState fetches the server's current state for a storage volume and
// updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r StorageVolumeResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m StorageVolumeModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	poolName := m.Pool.ValueString()
	volName := m.Name.ValueString()
	volType := m.Type.ValueString()
	vol, _, err := server.GetStoragePoolVolume(poolName, volType, volName)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage volume %q", volName), err.Error())
		return respDiags
	}

	// Extract user defined config and merge it with current config state.
	inheritedPoolVolumeKeys, err := m.InheritedStoragePoolVolumeKeys(server, poolName)
	if err != nil {
		respDiags.AddError(fmt.Sprintf("Failed to retrieve storage pool config %q", volName), err.Error())
		return respDiags
	}

	combinedComputedKeys := append(inheritedPoolVolumeKeys, m.ComputedKeys()...)
	stateConfig := common.StripConfig(vol.Config, m.Config, combinedComputedKeys)

	config, diags := common.ToConfigMapType(ctx, stateConfig, m.Config)
	respDiags.Append(diags...)

	m.Name = types.StringValue(vol.Name)
	m.Type = types.StringValue(vol.Type)
	m.Location = types.StringValue(vol.Location)
	m.Description = types.StringValue(vol.Description)
	m.ContentType = types.StringValue(vol.ContentType)
	m.Config = config

	m.Target = types.StringValue("")
	if server.IsClustered() || vol.Location != "none" {
		m.Target = types.StringValue(vol.Location)
	}

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

// ComputedKeys returns list of computed config keys.
func (StorageVolumeModel) ComputedKeys() []string {
	return []string{
		"block.filesystem",
		"block.mount_options",
		"volatile.",
	}
}

func (StorageVolumeModel) InheritedStoragePoolVolumeKeys(server incus.InstanceServer, poolName string) ([]string, error) {
	volumePrefix := "volume."
	inheritedKeys := make([]string, 0)

	pool, _, err := server.GetStoragePool(poolName)
	if err != nil {
		return nil, err
	}

	for key := range pool.Config {
		if strings.HasPrefix(key, volumePrefix) {
			inheritedKey := strings.TrimPrefix(key, volumePrefix)
			inheritedKeys = append(inheritedKeys, inheritedKey)
		}
	}

	return inheritedKeys, nil
}

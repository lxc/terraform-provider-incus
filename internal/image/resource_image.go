package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	"github.com/lxc/incus/v6/shared/archive"

	"github.com/lxc/terraform-provider-incus/internal/common"
	"github.com/lxc/terraform-provider-incus/internal/errors"
	provider_config "github.com/lxc/terraform-provider-incus/internal/provider-config"
	"github.com/lxc/terraform-provider-incus/internal/utils"
)

// ImageModel resource data model that matches the schema.
type ImageModel struct {
	SourceFile     types.Object `tfsdk:"source_file"`
	SourceImage    types.Object `tfsdk:"source_image"`
	SourceInstance types.Object `tfsdk:"source_instance"`
	Alias          types.Set    `tfsdk:"alias"`
	Project        types.String `tfsdk:"project"`
	Remote         types.String `tfsdk:"remote"`

	// Computed.
	ResourceID    types.String `tfsdk:"resource_id"`
	CreatedAt     types.Int64  `tfsdk:"created_at"`
	Fingerprint   types.String `tfsdk:"fingerprint"`
	CopiedAliases types.Set    `tfsdk:"copied_aliases"`
}

type SourceFileModel struct {
	DataPath     types.String `tfsdk:"data_path"`
	MetadataPath types.String `tfsdk:"metadata_path"`
}

type SourceImageModel struct {
	Remote       types.String `tfsdk:"remote"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	Architecture types.String `tfsdk:"architecture"`
	CopyAliases  types.Bool   `tfsdk:"copy_aliases"`
}

type SourceInstanceModel struct {
	Name     types.String `tfsdk:"name"`
	Snapshot types.String `tfsdk:"snapshot"`
}

type ImageAliasModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// ImageResource represent Incus cached image resource.
type ImageResource struct {
	provider *provider_config.IncusProviderConfig
}

// NewImageResource return new cached image resource.
func NewImageResource() resource.Resource {
	return &ImageResource{}
}

func (r ImageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_image", req.ProviderTypeName)
}

func (r ImageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"source_file": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"data_path": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"metadata_path": schema.StringAttribute{
						Optional: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
			},

			"source_image": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"remote": schema.StringAttribute{
						Required: true,
					},
					"name": schema.StringAttribute{
						Required: true,
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
					"architecture": schema.StringAttribute{
						Optional: true,
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							common.ArchitectureValidator{},
						},
					},
					"copy_aliases": schema.BoolAttribute{
						Optional: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.RequiresReplace(),
						},
					},
				},
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},

			"source_instance": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
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

			// Computed attributes.

			"resource_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"created_at": schema.Int64Attribute{
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},

			"fingerprint": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"copied_aliases": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},

		Blocks: map[string]schema.Block{
			"alias": schema.SetNestedBlock{
				Description: "Image alias",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the image alias.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"description": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The description for the image alias.",
							Default:     stringdefault.StaticString(""),
						},
					},
				},
			},
		},
	}
}

func (r *ImageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r ImageResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if req.Config.Raw.IsNull() {
		return
	}

	var config ImageModel

	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !exactlyOne(!config.SourceFile.IsNull(), !config.SourceImage.IsNull(), !config.SourceInstance.IsNull()) {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one of source_file, source_image or source_instance must be set.",
		)
		return
	}
}

func (r ImageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ImageModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.SourceFile.IsNull() {
		r.createImageFromSourceFile(ctx, resp, &plan)
		return
	} else if !plan.SourceImage.IsNull() {
		r.createImageFromSourceImage(ctx, resp, &plan)
		return
	} else if !plan.SourceInstance.IsNull() {
		r.createImageFromSourceInstance(ctx, resp, &plan)
		return
	}
}

func (r ImageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ImageModel

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
	diags = r.SyncState(ctx, &resp.State, server, state)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ImageModel

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

	// Extract image metadata.
	imageFingerprint := fingerprintFromResourceID(plan.ResourceID.ValueString())

	// Extract old and new nested alias blocks
	var oldImageAliasSet types.Set
	diags = req.State.GetAttribute(ctx, path.Root("alias"), &oldImageAliasSet)
	resp.Diagnostics.Append(diags...)

	oldImageAliases, diags := ToImageAliases(ctx, oldImageAliasSet)
	resp.Diagnostics.Append(diags...)

	newImageAliases, diags := ToImageAliases(ctx, plan.Alias)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	addedImageAliases, removedImageAliases := diffImageAliases(oldImageAliases, newImageAliases)

	diags = checkImageAliasesExist(server, addedImageAliases)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete removed aliases.
	for _, imageAlias := range removedImageAliases {
		err := server.DeleteImageAlias(imageAlias.Name)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete alias %q for cached image with fingerprint %q", imageAlias.Name, imageFingerprint), err.Error())
			return
		}
	}

	// Add new nested alias blocks (with descriptions)
	for _, imageAlias := range addedImageAliases {
		req := api.ImageAliasesPost{}
		req.Name = imageAlias.Name
		req.Description = imageAlias.Description
		req.Target = imageFingerprint

		err := server.CreateImageAlias(req)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to create alias %q for cached image with fingerprint %q", imageAlias.Name, imageFingerprint), err.Error())
			return
		}
	}

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, plan)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ImageModel

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

	imageFingerprint := fingerprintFromResourceID(state.ResourceID.ValueString())

	opDelete, err := server.DeleteImage(imageFingerprint)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cached image with fingerprint %q", imageFingerprint), err.Error())
		return
	}

	err = opDelete.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to remove cached image with fingerprint %q", imageFingerprint), err.Error())
		return
	}
}

// SyncState fetches the server's current state for a cached image and
// updates the provided model. It then applies this updated model as the
// new state in Terraform.
func (r ImageResource) SyncState(ctx context.Context, tfState *tfsdk.State, server incus.InstanceServer, m ImageModel) diag.Diagnostics {
	var respDiags diag.Diagnostics

	imageFingerprint := fingerprintFromResourceID(m.ResourceID.ValueString())

	image, _, err := server.GetImage(imageFingerprint)
	if err != nil {
		if errors.IsNotFoundError(err) {
			tfState.RemoveResource(ctx)
			return nil
		}

		respDiags.AddError(fmt.Sprintf("Failed to retrieve cached image with fingerprint %q", imageFingerprint), err.Error())
		return respDiags
	}

	if !m.SourceImage.IsNull() {
		var sourceImageModel SourceImageModel
		respDiags = m.SourceImage.As(ctx, &sourceImageModel, basetypes.ObjectAsOptions{})
		if respDiags.HasError() {
			return respDiags
		}

		// Store architecture if computed
		if sourceImageModel.Architecture.IsNull() || sourceImageModel.Architecture.IsUnknown() {
			sourceImageModel.Architecture = types.StringValue(image.Architecture)
			m.SourceImage, respDiags = types.ObjectValue(m.SourceImage.AttributeTypes(ctx), map[string]attr.Value{
				"remote":       sourceImageModel.Remote,
				"name":         sourceImageModel.Name,
				"type":         sourceImageModel.Type,
				"architecture": sourceImageModel.Architecture,
				"copy_aliases": sourceImageModel.CopyAliases,
			})
			if respDiags.HasError() {
				return respDiags
			}
		}
	}

	copiedAliases, diags := ToAliasList(ctx, m.CopiedAliases, func(alias string) string {
		return alias
	})
	respDiags.Append(diags...)

	configAliases, diags := ToAliasList(ctx, m.Alias, func(alias ImageAliasModel) string {
		return alias.Name.ValueString()
	})
	respDiags.Append(diags...)

	// Copy aliases from image state that are present in user defined
	// config or are not copied.
	var imageAliases []api.ImageAlias
	for _, a := range image.Aliases {
		if utils.ValueInSlice(a.Name, configAliases) || !utils.ValueInSlice(a.Name, copiedAliases) {
			imageAliases = append(imageAliases, a)
		}
	}

	copiedAliasesSet, diags := ToAliasSetType(ctx, copiedAliases)
	respDiags.Append(diags...)

	aliasBlockSet, diags := ToAliasBlockSetType(ctx, imageAliases)
	respDiags.Append(diags...)

	m.Fingerprint = types.StringValue(image.Fingerprint)
	m.CreatedAt = types.Int64Value(image.CreatedAt.Unix())
	m.CopiedAliases = copiedAliasesSet
	m.Alias = aliasBlockSet

	if respDiags.HasError() {
		return respDiags
	}

	return tfState.Set(ctx, &m)
}

func (r ImageResource) createImageFromSourceFile(ctx context.Context, resp *resource.CreateResponse, plan *ImageModel) {
	var sourceFileModel SourceFileModel

	diags := plan.SourceFile.As(ctx, &sourceFileModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	var dataPath, metadataPath string
	if sourceFileModel.MetadataPath.IsNull() {
		// Unified image
		metadataPath = sourceFileModel.DataPath.ValueString()
	} else {
		// Split image
		dataPath = sourceFileModel.DataPath.ValueString()
		metadataPath = sourceFileModel.MetadataPath.ValueString()
	}

	var image api.ImagesPost
	var createArgs *incus.ImageCreateArgs

	imageType := "container"
	if strings.HasPrefix(dataPath, "https://") {
		image.Source = &api.ImagesPostSource{}
		image.Source.Type = "url"
		image.Source.Mode = "pull"
		image.Source.Protocol = "direct"
		image.Source.URL = dataPath
		createArgs = nil
	} else {
		var meta io.ReadCloser
		var rootfs io.ReadCloser

		meta, err = os.Open(metadataPath)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to open metadata_path: %s", metadataPath), err.Error())
			return
		}

		defer func() { _ = meta.Close() }()

		// Open rootfs
		if dataPath != "" {
			rootfs, err = os.Open(dataPath)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("failed to open data_path: %s", dataPath), err.Error())
				return
			}

			defer func() { _ = rootfs.Close() }()

			_, ext, _, err := archive.DetectCompressionFile(rootfs)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to detect compression of rootfs in data_path: %s", dataPath), err.Error())
				return
			}

			_, err = rootfs.(*os.File).Seek(0, io.SeekStart)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to seek start for rootfas in data_path: %s", dataPath), err.Error())
				return
			}

			if ext == ".qcow2" {
				imageType = "virtual-machine"
			}
		}

		createArgs = &incus.ImageCreateArgs{
			MetaFile:   meta,
			MetaName:   filepath.Base(metadataPath),
			RootfsFile: rootfs,
			RootfsName: filepath.Base(dataPath),
			Type:       imageType,
		}

		image.Filename = createArgs.MetaName
	}

	imageAliases, diags := ToImageAliases(ctx, plan.Alias)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = checkImageAliasesExist(server, imageAliases)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	image.Aliases = imageAliases

	op, err := server.CreateImage(image, createArgs)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create image from file %q", dataPath), err.Error())
		return
	}

	// Wait for image create operation to finish.
	err = op.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to create image from file %q", dataPath), err.Error())
		return
	}

	fingerprint, ok := op.Get().Metadata["fingerprint"].(string)
	if !ok {
		resp.Diagnostics.AddError("Failed to get fingerprint of created image", "no fingerprint returned in metadata")
		return
	}
	imageID := createImageResourceID(remote, fingerprint)
	plan.ResourceID = types.StringValue(imageID)

	plan.CopiedAliases = basetypes.NewSetNull(basetypes.StringType{})

	diags = r.SyncState(ctx, &resp.State, server, *plan)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) createImageFromSourceImage(ctx context.Context, resp *resource.CreateResponse, plan *ImageModel) {
	var sourceImageModel SourceImageModel

	diags := plan.SourceImage.As(ctx, &sourceImageModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	image := sourceImageModel.Name.ValueString()
	imageType := sourceImageModel.Type.ValueString()
	imageRemote := sourceImageModel.Remote.ValueString()
	imageServer, err := r.provider.ImageServer(imageRemote)
	if err != nil {
		resp.Diagnostics.Append(errors.NewImageServerError(err))
		return
	}

	connInfo, err := imageServer.GetConnectionInfo()
	if err != nil {
		resp.Diagnostics.AddError("Failed to retrieve server connection info", err.Error())
		return
	}

	// Determine the correct image for the specified architecture.
	architecture := sourceImageModel.Architecture.ValueString()
	if architecture != "" {
		availableArchitectures, err := imageServer.GetImageAliasArchitectures(imageType, image)
		if err != nil {
			resp.Diagnostics.AddError("Failed to get image alias architectures", err.Error())
			return
		}

		found := false
		for imageArchitecture, imageAlias := range availableArchitectures {
			if imageArchitecture == architecture {
				image = imageAlias.Target
				found = true
			}
		}

		if !found {
			keys := make([]string, 0, len(availableArchitectures))
			for key := range availableArchitectures {
				keys = append(keys, key)
			}
			keyList := strings.Join(keys, ", ")

			resp.Diagnostics.AddError(fmt.Sprintf("No image alias found for architecture: %s. Available architectures: %s ", architecture, keyList), "")
			return
		}
	}

	// Determine whether the user has provided a fingerprint or an alias.
	aliasTarget, _, err := imageServer.GetImageAliasType(imageType, image)
	if connInfo.Protocol == "oci" && err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to get image alias for OCI image %s", image), err.Error())
		return
	}

	if aliasTarget != nil {
		image = aliasTarget.Target
	}

	imageAliases, diags := ToImageAliases(ctx, plan.Alias)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = checkImageAliasesExist(server, imageAliases)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Get data about remote image (also checks if image exists).
	imageInfo, _, err := imageServer.GetImage(image)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve info about image %q", image), err.Error())
		return
	}

	// Copy image.
	args := incus.ImageCopyArgs{
		Aliases: imageAliases,
		Public:  false,
	}

	var opCopy incus.RemoteOperation
	if connInfo.Protocol == "oci" {
		// For OCI images, we need to use the actual image name to pull the image from the current OCI images registry.
		// Nevertheless, we need to restore the actual fingerprint after copying the image by name.
		realFingerprint := imageInfo.Fingerprint
		imageInfo.Fingerprint = sourceImageModel.Name.ValueString()
		opCopy, err = server.CopyImage(imageServer, *imageInfo, &args)
		imageInfo.Fingerprint = realFingerprint
	} else {
		opCopy, err = server.CopyImage(imageServer, *imageInfo, &args)
	}

	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy image %q", image), err.Error())
		return
	}

	// Wait for copy operation to finish.
	err = opCopy.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to copy image %q", image), err.Error())
		return
	}

	// Store remote aliases that we've copied, so we can filter them
	// out later.
	copied := make([]string, 0)
	if sourceImageModel.CopyAliases.ValueBool() {
		for _, a := range imageInfo.Aliases {
			copied = append(copied, a.Name)
		}
	}

	copiedAliases, diags := types.SetValueFrom(ctx, types.StringType, copied)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	imageID := createImageResourceID(remote, imageInfo.Fingerprint)
	plan.ResourceID = types.StringValue(imageID)

	plan.CopiedAliases = copiedAliases

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, *plan)
	resp.Diagnostics.Append(diags...)
}

func (r ImageResource) createImageFromSourceInstance(ctx context.Context, resp *resource.CreateResponse, plan *ImageModel) {
	var sourceInstanceModel SourceInstanceModel

	diags := plan.SourceInstance.As(ctx, &sourceInstanceModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	remote := plan.Remote.ValueString()
	project := plan.Project.ValueString()
	server, err := r.provider.InstanceServer(remote, project, "")
	if err != nil {
		resp.Diagnostics.Append(errors.NewInstanceServerError(err))
		return
	}

	instanceName := sourceInstanceModel.Name.ValueString()
	instanceState, _, err := server.GetInstanceState(instanceName)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to retrieve state of instance %q", instanceName), err.Error())
		return
	}

	if sourceInstanceModel.Snapshot.IsNull() && instanceState.StatusCode != api.Stopped {
		resp.Diagnostics.AddError(fmt.Sprintf("Cannot publish image because instance %q is running", instanceName), "")
		return
	}

	imageAliases, diags := ToImageAliases(ctx, plan.Alias)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = checkImageAliasesExist(server, imageAliases)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var source *api.ImagesPostSource
	if !sourceInstanceModel.Snapshot.IsNull() {
		snapsnotName := sourceInstanceModel.Snapshot.ValueString()
		source = &api.ImagesPostSource{
			Name: fmt.Sprintf("%s/%s", instanceName, snapsnotName),
			Type: "snapshot",
		}
	} else {
		source = &api.ImagesPostSource{
			Name: instanceName,
			Type: "instance",
		}
	}

	imageReq := api.ImagesPost{
		Aliases:  imageAliases,
		ImagePut: api.ImagePut{},
		Source:   source,
	}

	// Publish image.
	op, err := server.CreateImage(imageReq, nil)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to publish instance %q image", instanceName), err.Error())
		return
	}

	// Wait for create operation to finish.
	err = op.Wait()
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Failed to publish instance %q image", instanceName), err.Error())
		return
	}

	// Extract fingerprint from operation response.
	opResp := op.Get()
	imageFingerprint, ok := opResp.Metadata["fingerprint"].(string)
	if !ok {
		resp.Diagnostics.AddError(fmt.Sprintf(`Fingerprint "%v" is not a string`, opResp.Metadata["fingerprint"]), fmt.Sprintf(`Fingerprint "%[1]v" is not a string but %[1]T`, opResp.Metadata["fingerprint"]))
		return
	}

	plan.Fingerprint = types.StringValue(imageFingerprint)

	imageID := createImageResourceID(remote, imageFingerprint)
	plan.ResourceID = types.StringValue(imageID)

	plan.CopiedAliases = types.SetNull(types.StringType)

	// Update Terraform state.
	diags = r.SyncState(ctx, &resp.State, server, *plan)
	resp.Diagnostics.Append(diags...)
}

// ToAliasList converts aliases of type types.Set into a slice of strings.
func ToAliasList[T any](ctx context.Context, aliasSet types.Set, converter func(T) string) ([]string, diag.Diagnostics) {
	if aliasSet.IsNull() || aliasSet.IsUnknown() {
		return []string{}, nil
	}

	elements := make([]T, 0, len(aliasSet.Elements()))
	diags := aliasSet.ElementsAs(ctx, &elements, false)
	if diags.HasError() {
		return nil, diags
	}

	aliases := make([]string, 0, len(elements))
	for _, element := range elements {
		aliases = append(aliases, converter(element))
	}

	return aliases, diags
}

// ToAliasSetType converts slice of strings into aliases of type types.Set.
func ToAliasSetType(ctx context.Context, aliases []string) (types.Set, diag.Diagnostics) {
	return types.SetValueFrom(ctx, types.StringType, aliases)
}

// ToAliasBlockSetType converts slice of api.ImageAlias into alias block of type types.Set.
func ToAliasBlockSetType(ctx context.Context, aliases []api.ImageAlias) (types.Set, diag.Diagnostics) {
	aliasList := make([]ImageAliasModel, 0, len(aliases))

	for _, a := range aliases {
		alias := ImageAliasModel{
			Name:        types.StringValue(a.Name),
			Description: types.StringValue(a.Description),
		}

		aliasList = append(aliasList, alias)
	}

	aliasType := map[string]attr.Type{
		"name":        types.StringType,
		"description": types.StringType,
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: aliasType}, aliasList)
}

// ToAliasModelList converts image alias blocks from types.Set into
// a list of ImageAliasModel.
func ToImageAliases(ctx context.Context, aliasSet types.Set) ([]api.ImageAlias, diag.Diagnostics) {
	if aliasSet.IsNull() || aliasSet.IsUnknown() {
		return []api.ImageAlias{}, nil
	}

	aliasModelList := make([]ImageAliasModel, 0, len(aliasSet.Elements()))
	diags := aliasSet.ElementsAs(ctx, &aliasModelList, false)
	if diags.HasError() {
		return nil, diags
	}

	imageAliases := make([]api.ImageAlias, 0, len(aliasModelList))
	for _, aliasModel := range aliasModelList {
		imageAlias := api.ImageAlias{
			Name:        aliasModel.Name.ValueString(),
			Description: aliasModel.Description.ValueString(),
		}

		imageAliases = append(imageAliases, imageAlias)
	}

	return imageAliases, diags
}

func diffImageAliases(oldImageAliases, newImageAliases []api.ImageAlias) (added, removed []api.ImageAlias) {
	oldImageAliasMap := make(map[string]api.ImageAlias)
	for _, imageAlias := range oldImageAliases {
		oldImageAliasMap[imageAlias.Name] = imageAlias
	}

	newImageAliasMap := make(map[string]api.ImageAlias)
	for _, imageAlias := range newImageAliases {
		newImageAliasMap[imageAlias.Name] = imageAlias
	}

	for name, imageAlias := range newImageAliasMap {
		if _, exists := oldImageAliasMap[name]; !exists {
			added = append(added, imageAlias)
		}
	}

	for name, imageAlias := range oldImageAliasMap {
		if _, exists := newImageAliasMap[name]; !exists {
			removed = append(removed, imageAlias)
		}
	}

	return added, removed
}

// check image aliases existence by name.
func checkImageAliasesExist(server incus.InstanceServer, imageAliases []api.ImageAlias) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, imageAlias := range imageAliases {
		// Ensure image alias does not already exist.
		aliasTarget, _, _ := server.GetImageAlias(imageAlias.Name)
		if aliasTarget != nil {
			diags.AddError(fmt.Sprintf("Image alias %q already exists", imageAlias.Name), "")
			return diags
		}
	}

	return nil
}

// createImageResourceID creates new image ID by concatenating remote and
// image fingerprint using colon.
func createImageResourceID(remote string, fingerprint string) string {
	return fmt.Sprintf("%s:%s", remote, fingerprint)
}

// fingerprintFromResourceID returns the fingerprint part from an image ID.
func fingerprintFromResourceID(id string) string {
	pieces := strings.SplitN(id, ":", 2)
	return pieces[1]
}

func exactlyOne(in ...bool) bool {
	var count int
	for _, b := range in {
		if b {
			count++
		}
	}
	return count == 1
}

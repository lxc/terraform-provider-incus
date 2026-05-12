package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfpath "github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/mitchellh/go-homedir"

	tfierrors "github.com/lxc/terraform-provider-incus/internal/errors"
)

type InstanceFileModel struct {
	Content         types.String `tfsdk:"content"`
	SourcePath      types.String `tfsdk:"source_path"`
	TargetPath      types.String `tfsdk:"target_path"`
	UserID          types.Int64  `tfsdk:"uid"`
	GroupID         types.Int64  `tfsdk:"gid"`
	Mode            types.String `tfsdk:"mode"`
	CreateDirs      types.Bool   `tfsdk:"create_directories"`
	Append          types.Bool   `tfsdk:"append"`
	ContentHash     types.String `tfsdk:"content_hash"`
}

// ToFileMap converts files from types.Set into map[string]IncusFileModel.
func ToFileMap(ctx context.Context, fileSet types.Set) (map[string]InstanceFileModel, diag.Diagnostics) {
	if fileSet.IsNull() || fileSet.IsUnknown() {
		return make(map[string]InstanceFileModel), nil
	}

	files := make([]InstanceFileModel, 0, len(fileSet.Elements()))
	diags := fileSet.ElementsAs(ctx, &files, false)
	if diags.HasError() {
		return nil, diags
	}

	// Convert list into map.
	fileMap := make(map[string]InstanceFileModel, len(files))
	for _, f := range files {
		fileMap[f.TargetPath.ValueString()] = f
	}

	return fileMap, diags
}

// fileTypeAttrTypes defines the attribute types for the file nested block.
// This must match the InstanceFileModel struct fields and the schema definition.
var fileTypeAttrTypes = map[string]attr.Type{
	"content":            types.StringType,
	"source_path":        types.StringType,
	"target_path":        types.StringType,
	"uid":                types.Int64Type,
	"gid":                types.Int64Type,
	"mode":               types.StringType,
	"create_directories": types.BoolType,
	"append":             types.BoolType,
	"content_hash":       types.StringType,
}

// ToFileSetType converts files from a map[string]IncusFileModel into types.Set.
func ToFileSetType(ctx context.Context, fileMap map[string]InstanceFileModel) (types.Set, diag.Diagnostics) {
	files := make([]InstanceFileModel, 0, len(fileMap))
	for _, v := range fileMap {
		files = append(files, v)
	}

	return types.SetValueFrom(ctx, types.ObjectType{AttrTypes: fileTypeAttrTypes}, files)
}

// InstanceFileDelete deletes a file from an instance.
func InstanceFileDelete(server incus.InstanceServer, instanceName string, targetPath string) error {
	err := server.DeleteInstanceFile(instanceName, targetPath)
	if err != nil && !tfierrors.IsNotFoundError(err) {
		return err
	}

	return nil
}

// InstanceFileUpload uploads a file to an instance.
// It computes a SHA256 hash of the uploaded content and sets it on the model.
func InstanceFileUpload(server incus.InstanceServer, instanceName string, file *InstanceFileModel) error {
	content := file.Content.ValueString()
	sourcePath := file.SourcePath.ValueString()

	if content != "" && sourcePath != "" {
		return fmt.Errorf("File %q and %q are mutually exclusive.", "content", "source_path")
	}

	targetPath := file.TargetPath.ValueString()

	fileMode := file.Mode.ValueString()
	if fileMode == "" {
		fileMode = "0755"
	}

	mode, err := strconv.ParseUint(fileMode, 8, 32)
	if err != nil {
		return fmt.Errorf("Failed to parse file mode: %v", err)
	}

	// Build the file creation request, without the content.
	args := &incus.InstanceFileArgs{
		Type: "file",
		Mode: int(mode),
		UID:  file.UserID.ValueInt64(),
		GID:  file.GroupID.ValueInt64(),
	}

	if file.Append.ValueBool() {
		args.WriteMode = "append"
	} else {
		args.WriteMode = "overwrite"
	}

	// Hash will be computed from the content being uploaded.
	var contentHash string

	// If content was specified, read the string.
	if content != "" {
		contentHash = ComputeFileHash(content)
		args.Content = strings.NewReader(content)
	}

	// If a source was specified, read the contents of the source file.
	if sourcePath != "" {
		path, err := homedir.Expand(sourcePath)
		if err != nil {
			return fmt.Errorf("Unable to determine source file path: %v", err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("Unable to read source file: %v", err)
		}
		defer func(f *os.File) {
			err = errors.Join(err, f.Close())
		}(f)

		// Read file content for hashing, then wrap in a reader for upload.
		fileBytes, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("Unable to read source file content: %v", err)
		}
		contentHash = ComputeFileHash(string(fileBytes))
		args.Content = strings.NewReader(string(fileBytes))
	}

	if file.CreateDirs.ValueBool() {
		err := instanceRecursiveMkdir(server, instanceName, path.Dir(targetPath), *args)
		if err != nil {
			return fmt.Errorf("Could not create directories for file %q: %v", targetPath, err)
		}
	}

	err = server.CreateInstanceFile(instanceName, targetPath, *args)
	if err != nil {
		return fmt.Errorf("Could not upload file %q: %v", targetPath, err)
	}

	// Store the computed hash on the model.
	file.ContentHash = types.StringValue(contentHash)

	return nil
}

// instanceRecursiveMkdir recursively creates directories on target instance.
func instanceRecursiveMkdir(server incus.InstanceServer, instanceName string, p string, args incus.InstanceFileArgs) error {
	// Special case, every instance has a /, so there is nothing to do.
	if p == "/" {
		return nil
	}

	// Remove trailing "/" e.g. /A/B/C/. Otherwise we will end up with an
	// empty array entry "" which will confuse the Mkdir() loop below.
	pclean := filepath.Clean(p)
	parts := strings.Split(pclean, "/")
	i := len(parts)

	for ; i >= 1; i-- {
		cur := filepath.Join(parts[:i]...)
		_, resp, err := server.GetInstanceFile(instanceName, cur)
		if err != nil {
			continue
		}

		if resp.Type != "directory" {
			return fmt.Errorf("%s is not a directory", cur)
		}

		i++
		break
	}

	// Use same arguments as for file upload, only change file type.
	dirArgs := incus.InstanceFileArgs{
		Type: "directory",
		Mode: args.Mode,
		UID:  args.UID,
		GID:  args.GID,
	}

	for ; i <= len(parts); i++ {
		cur := filepath.Join(parts[:i]...)
		if cur == "" {
			continue
		}

		err := server.CreateInstanceFile(instanceName, cur, dirArgs)
		if err != nil {
			return err
		}
	}

	return nil
}

// VolumeFileDelete deletes a file from a storage volume.
func VolumeFileDelete(server incus.InstanceServer, pool, volumeType, volumeName, targetPath string) error {
	err := server.DeleteStorageVolumeFile(pool, volumeType, volumeName, targetPath)
	if err != nil && !tfierrors.IsNotFoundError(err) {
		return err
	}

	return nil
}

// VolumeFileUpload uploads a file to a storage volume.
func VolumeFileUpload(server incus.InstanceServer, pool, volumeType, volumeName string, file *InstanceFileModel) error {
	content := file.Content.ValueString()
	sourcePath := file.SourcePath.ValueString()

	if content != "" && sourcePath != "" {
		return fmt.Errorf("File %q and %q are mutually exclusive.", "content", "source_path")
	}

	targetPath := file.TargetPath.ValueString()

	fileMode := file.Mode.ValueString()
	if fileMode == "" {
		fileMode = "0755"
	}

	mode, err := strconv.ParseUint(fileMode, 8, 32)
	if err != nil {
		return fmt.Errorf("Failed to parse file mode: %v", err)
	}

	// Build the file creation request, without the content.
	args := &incus.InstanceFileArgs{
		Type: "file",
		Mode: int(mode),
		UID:  file.UserID.ValueInt64(),
		GID:  file.GroupID.ValueInt64(),
	}

	if file.Append.ValueBool() {
		args.WriteMode = "append"
	} else {
		args.WriteMode = "overwrite"
	}

	// Hash will be computed from the content being uploaded.
	var contentHash string

	// If content was specified, read the string.
	if content != "" {
		contentHash = ComputeFileHash(content)
		args.Content = strings.NewReader(content)
	}

	// If a source was specified, read the contents of the source file.
	if sourcePath != "" {
		currentPath, err := homedir.Expand(sourcePath)
		if err != nil {
			return fmt.Errorf("Unable to determine source file currentPath: %v", err)
		}

		f, err := os.Open(currentPath)
		if err != nil {
			return fmt.Errorf("Unable to read source file: %v", err)
		}
		defer func(f *os.File) {
			err = errors.Join(err, f.Close())
		}(f)

		// Read file content for hashing, then wrap in a reader for upload.
		fileBytes, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("Unable to read source file content: %v", err)
		}
		contentHash = ComputeFileHash(string(fileBytes))
		args.Content = strings.NewReader(string(fileBytes))
	}

	if file.CreateDirs.ValueBool() {
		err := volumeRecursiveMkdir(server, pool, volumeType, volumeName, path.Dir(targetPath), *args)
		if err != nil {
			return fmt.Errorf("Could not create directories for file %q: %v", targetPath, err)
		}
	}

	err = server.CreateStorageVolumeFile(pool, volumeType, volumeName, targetPath, *args)
	if err != nil {
		return fmt.Errorf("Could not upload file %q: %v", targetPath, err)
	}

	// Store the computed hash on the model.
	file.ContentHash = types.StringValue(contentHash)

	return nil
}

// volumeRecursiveMkdir recursively creates directories on target instance.
func volumeRecursiveMkdir(server incus.InstanceServer, pool, volumeType, volumeName, p string, args incus.InstanceFileArgs) error {
	// Special case, every instance has a /, so there is nothing to do.
	if p == "/" {
		return nil
	}

	// Remove trailing "/" e.g. /A/B/C/. Otherwise we will end up with an
	// empty array entry "" which will confuse the Mkdir() loop below.
	pclean := filepath.Clean(p)
	parts := strings.Split(pclean, "/")
	i := len(parts)

	for ; i >= 1; i-- {
		cur := filepath.Join(parts[:i]...)
		_, resp, err := server.GetStorageVolumeFile(pool, volumeType, volumeName, cur)
		if err != nil {
			continue
		}

		if resp.Type != "directory" {
			return fmt.Errorf("%s is not a directory", cur)
		}

		i++
		break
	}

	// Use same arguments as for file upload, only change file type.
	dirArgs := incus.InstanceFileArgs{
		Type: "directory",
		Mode: args.Mode,
		UID:  args.UID,
		GID:  args.GID,
	}

	for ; i <= len(parts); i++ {
		cur := filepath.Join(parts[:i]...)
		if cur == "" {
			continue
		}

		err := server.CreateStorageVolumeFile(pool, volumeType, volumeName, cur, dirArgs)
		if err != nil {
			return err
		}
	}

	return nil
}

func HasFileContentChanged(newFile InstanceFileModel, oldFile InstanceFileModel) bool {
	// If both files have content hashes, compare those first.
	hasNewHash := !newFile.ContentHash.IsNull() && !newFile.ContentHash.IsUnknown()
	hasOldHash := !oldFile.ContentHash.IsNull() && !oldFile.ContentHash.IsUnknown()

	if hasNewHash && hasOldHash {
		return newFile.ContentHash.ValueString() != oldFile.ContentHash.ValueString()
	}

	// Fall back to content/source_path comparison.
	hasNewContent := !newFile.Content.IsNull()
	hasOldContent := !oldFile.Content.IsNull()
	hasNewSourcePath := !newFile.SourcePath.IsNull()
	hasOldSourcePath := !oldFile.SourcePath.IsNull()

	switch {
	case hasNewContent && hasOldContent:
		return newFile.Content.ValueString() != oldFile.Content.ValueString()
	case hasNewSourcePath && hasOldSourcePath:
		return newFile.SourcePath.ValueString() != oldFile.SourcePath.ValueString()
	case (hasNewSourcePath && hasOldContent) || (hasNewContent && hasOldSourcePath):
		return true
	}

	return false
}

func HasFilePermissionChanged(newFile InstanceFileModel, oldFile InstanceFileModel) bool {
	return newFile.Mode.ValueString() != oldFile.Mode.ValueString() ||
		newFile.UserID.ValueInt64() != oldFile.UserID.ValueInt64() ||
		newFile.GroupID.ValueInt64() != oldFile.GroupID.ValueInt64()
}

// ModifyPlanFileHashes computes content_hash for each file in the plan during
// the plan phase. This avoids "inconsistent result after apply" errors caused
// by computed attributes inside SetNestedBlock, where Terraform cannot correlate
// planned elements (with unknown computed values) to applied elements (with known values).
func ModifyPlanFileHashes(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	var planFiles types.Set
	diags := req.Plan.GetAttribute(ctx, tfpath.Root("file"), &planFiles)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() || planFiles.IsNull() || planFiles.IsUnknown() {
		return
	}

	fileMap, diags := ToFileMap(ctx, planFiles)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	for targetPath, file := range fileMap {
		if !file.ContentHash.IsNull() && !file.ContentHash.IsUnknown() {
			continue
		}

		content := file.Content.ValueString()
		sourcePath := file.SourcePath.ValueString()

		if content != "" {
			file.ContentHash = types.StringValue(ComputeFileHash(content))
		} else if sourcePath != "" {
			expandedPath, err := homedir.Expand(sourcePath)
			if err != nil {
				continue
			}
			fileBytes, err := os.ReadFile(expandedPath)
			if err != nil {
				continue
			}
			file.ContentHash = types.StringValue(ComputeFileHash(string(fileBytes)))
		}

		fileMap[targetPath] = file
	}

	updatedFiles, diags := ToFileSetType(ctx, fileMap)
	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	resp.Plan.SetAttribute(ctx, tfpath.Root("file"), updatedFiles)
}

// ComputeFileHash computes a SHA256 hash of content and returns the hex-encoded string.
func ComputeFileHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// FileModeToString converts an integer file mode to an octal permission string (e.g., 493 -> "0755").
// It masks to permission bits only (0o777) since the server response includes file type bits.
func FileModeToString(mode int) string {
	return fmt.Sprintf("%04o", mode&0o777)
}

// InstanceFileRead reads a file's content and metadata from an instance.
// Returns the file content as a string and the file response metadata.
func InstanceFileRead(server incus.InstanceServer, instanceName string, targetPath string) (string, *incus.InstanceFileResponse, error) {
	reader, fileResp, err := server.GetInstanceFile(instanceName, targetPath)
	if err != nil {
		return "", nil, fmt.Errorf("Failed to read file %q from instance %q: %w", targetPath, instanceName, err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, fmt.Errorf("Failed to read content of file %q from instance %q: %w", targetPath, instanceName, err)
	}

	return string(content), fileResp, nil
}

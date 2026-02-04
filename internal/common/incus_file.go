package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/mitchellh/go-homedir"

	tfierrors "github.com/lxc/terraform-provider-incus/internal/errors"
)

type InstanceFileModel struct {
	Content    types.String `tfsdk:"content"`
	SourcePath types.String `tfsdk:"source_path"`
	TargetPath types.String `tfsdk:"target_path"`
	UserID     types.Int64  `tfsdk:"uid"`
	GroupID    types.Int64  `tfsdk:"gid"`
	Mode       types.String `tfsdk:"mode"`
	CreateDirs types.Bool   `tfsdk:"create_directories"`
	Append     types.Bool   `tfsdk:"append"`
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

// ToFileSetType converts files from a map[string]IncusFileModel into types.Set.
func ToFileSetType(ctx context.Context, fileMap map[string]InstanceFileModel) (types.Set, diag.Diagnostics) {
	files := make([]InstanceFileModel, 0, len(fileMap))
	for _, v := range fileMap {
		files = append(files, v)
	}

	return types.SetValueFrom(ctx, types.ObjectType{}, files)
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
func InstanceFileUpload(server incus.InstanceServer, instanceName string, file InstanceFileModel) error {
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

	// If content was specified, read the string.
	if content != "" {
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

		args.Content = f
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
func VolumeFileUpload(server incus.InstanceServer, pool, volumeType, volumeName string, file InstanceFileModel) error {
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

	// If content was specified, read the string.
	if content != "" {
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

		args.Content = f
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

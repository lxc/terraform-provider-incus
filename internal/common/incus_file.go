package common

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

// ToFileMap converts files from types.Set into map[string]InstanceFileModel.
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

// ToFileSetType converts files from a map[string]InstanceFileModel into types.Set.
func ToFileSetType(ctx context.Context, fileMap map[string]InstanceFileModel) (types.Set, diag.Diagnostics) {
	files := make([]InstanceFileModel, 0, len(fileMap))
	for _, v := range fileMap {
		files = append(files, v)
	}

	return types.SetValueFrom(ctx, types.ObjectType{}, files)
}

// coreFileDelete deletes a file from a resource (either an instance or a volume).
func coreFileDelete(targetPath string, deleteOperation func(targetPath string) error) error {
	err := deleteOperation(targetPath)
	if err != nil && !tfierrors.IsNotFoundError(err) {
		return err
	}

	return nil
}

func InstanceCreateFileOperation(server incus.InstanceServer, instanceName string) func(string, incus.InstanceFileArgs) error {
	return func(targetPath string, args incus.InstanceFileArgs) error {
		return server.CreateInstanceFile(instanceName, targetPath, args)
	}
}

func InstanceGetFileOperation(server incus.InstanceServer, instanceName string) func(string) (io.ReadCloser, *incus.InstanceFileResponse, error) {
	return func(targetPath string) (io.ReadCloser, *incus.InstanceFileResponse, error) {
		return server.GetInstanceFile(instanceName, targetPath)
	}
}

func InstanceDeleteFileOperation(server incus.InstanceServer, instanceName string) func(string) error {
	return func(targetPath string) error {
		return server.DeleteInstanceFile(instanceName, targetPath)
	}
}

// InstanceFileUpload uploads a file to an instance.
func InstanceFileUpload(server incus.InstanceServer, instanceName string, file InstanceFileModel) error {
	createOperation := InstanceCreateFileOperation(server, instanceName)
	getOperation := InstanceGetFileOperation(server, instanceName)
	return coreFileUpload(file, createOperation, getOperation)
}

func VolumeCreateFileOperation(server incus.InstanceServer, pool, volumeType, volumeName string) func(string, incus.InstanceFileArgs) error {
	return func(targetPath string, args incus.InstanceFileArgs) error {
		return server.CreateStorageVolumeFile(pool, volumeType, volumeName, targetPath, args)
	}
}

func VolumeGetFileOperation(server incus.InstanceServer, pool, volumeType, volumeName string) func(string) (io.ReadCloser, *incus.InstanceFileResponse, error) {
	return func(targetPath string) (io.ReadCloser, *incus.InstanceFileResponse, error) {
		return server.GetStorageVolumeFile(pool, volumeType, volumeName, targetPath)
	}
}

func VolumeDeleteFileOperation(server incus.InstanceServer, pool, volumeType, volumeName string) func(string) error {
	return func(targetPath string) error {
		return server.DeleteStorageVolumeFile(pool, volumeType, volumeName, targetPath)
	}
}

// VolumeFileUpload uploads a file to an volume.
func VolumeFileUpload(server incus.InstanceServer, pool, volumeType, volumeName string, file InstanceFileModel) error {
	createOperation := VolumeCreateFileOperation(server, pool, volumeType, volumeName)
	getOperation := VolumeGetFileOperation(server, pool, volumeType, volumeName)
	return coreFileUpload(file, createOperation, getOperation)
}

// coreFileUpload uploads a file to a resource (either an instance or a volume).
func coreFileUpload(file InstanceFileModel, createOperation func(string, incus.InstanceFileArgs) error, getOperation func(string) (io.ReadCloser, *incus.InstanceFileResponse, error)) (err error) {
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
			return fmt.Errorf("Unable to determine source file path: %v", err)
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
		err := recursiveMkdir(path.Dir(targetPath), *args, createOperation, getOperation)
		if err != nil {
			return fmt.Errorf("Could not create directories for file %q: %v", targetPath, err)
		}
	}

	err = createOperation(targetPath, *args)
	if err != nil {
		return fmt.Errorf("Could not upload file %q: %v", targetPath, err)
	}

	return nil
}

// recursiveMkdir recursively creates directories on target resource.
// This was copied almost as-is from github.com/lxc/incus/blob/main/lxc/file.go.
func recursiveMkdir(p string, args incus.InstanceFileArgs, createOperation func(string, incus.InstanceFileArgs) error, getOperation func(string) (io.ReadCloser, *incus.InstanceFileResponse, error)) error {
	// Special case, every resource has a /, so there is nothing to do.
	if p == "/" {
		return nil
	}

	// Remove trailing "/" e.g. /A/B/C/. Otherwise, we will end up with an
	// empty array entry "" which will confuse the Mkdir() loop below.
	pclean := filepath.Clean(p)
	parts := strings.Split(pclean, "/")
	i := len(parts)

	for ; i >= 1; i-- {
		cur := filepath.Join(parts[:i]...)
		_, resp, err := getOperation(cur)
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

		err := createOperation(cur, dirArgs)
		if err != nil {
			return err
		}
	}

	return nil
}

func hasFileContentChanged(newFile InstanceFileModel, oldFile InstanceFileModel) bool {
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

func hasFilePermissionChanged(newFile InstanceFileModel, oldFile InstanceFileModel) bool {
	return newFile.Mode.ValueString() != oldFile.Mode.ValueString() ||
		newFile.UserID.ValueInt64() != oldFile.UserID.ValueInt64() ||
		newFile.GroupID.ValueInt64() != oldFile.GroupID.ValueInt64()
}

func UpdateFiles(ctx context.Context, targetResource string, stateFiles, planFiles types.Set, resp *resource.UpdateResponse, createOperation func(string, incus.InstanceFileArgs) error, getOperation func(string) (io.ReadCloser, *incus.InstanceFileResponse, error), deleteOperation func(targetPath string) error) {
	oldFiles, diags := ToFileMap(ctx, stateFiles)
	resp.Diagnostics.Append(diags...)

	newFiles, diags := ToFileMap(ctx, planFiles)
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
		err := coreFileDelete(targetPath, deleteOperation)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file from %q", targetResource), err.Error())
			return
		}
	}

	// Upload new files or update existing files if content has changed.
	for k, newFile := range newFiles {
		oldFile, exists := oldFiles[k]

		if !exists {
			err := coreFileUpload(newFile, createOperation, getOperation)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload file to %q", targetResource), err.Error())
				return
			}
			continue
		}

		contentChanged := hasFileContentChanged(newFile, oldFile)
		permissionsChanged := hasFilePermissionChanged(newFile, oldFile)

		if contentChanged || permissionsChanged {
			// Delete the old file first otherwise mode and ownership changes
			// will not be applied.
			targetPath := newFile.TargetPath.ValueString()
			err := deleteOperation(targetPath)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to delete file from %q", targetResource), err.Error())
				return
			}

			err = coreFileUpload(newFile, createOperation, getOperation)
			if err != nil {
				resp.Diagnostics.AddError(fmt.Sprintf("Failed to upload updated file to %q", targetResource), err.Error())
				return
			}
		}
	}
}

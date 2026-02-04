package common

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	incus "github.com/lxc/incus/v6/client"
	"github.com/lxc/incus/v6/shared/api"
)

type InstanceExecModel struct {
	Command     types.List   `tfsdk:"command"`
	Environment types.Map    `tfsdk:"environment"`
	WorkingDir  types.String `tfsdk:"working_dir"`
	UserID      types.Int64  `tfsdk:"uid"`
	GroupID     types.Int64  `tfsdk:"gid"`
	Timeout     types.String `tfsdk:"timeout"`
	Trigger     types.String `tfsdk:"trigger"`
}

type InstanceExecConfig struct {
	Command     []string
	Environment map[string]string
	WorkingDir  string
	UserID      int64
	GroupID     int64
	HasUserID   bool
	HasGroupID  bool
	Timeout     time.Duration
	HasTimeout  bool
	Trigger     string
}

func ToExecMap(ctx context.Context, execMap types.Map) (map[string]InstanceExecModel, diag.Diagnostics) {
	if execMap.IsNull() || execMap.IsUnknown() {
		return make(map[string]InstanceExecModel), nil
	}

	execs := make(map[string]InstanceExecModel, len(execMap.Elements()))
	diags := execMap.ElementsAs(ctx, &execs, false)
	if diags.HasError() {
		return nil, diags
	}

	return execs, nil
}

func ToExecConfig(ctx context.Context, exec InstanceExecModel) (InstanceExecConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	execConfig := InstanceExecConfig{
		Environment: map[string]string{},
		Trigger:     "on_change",
	}

	if !exec.Command.IsNull() && !exec.Command.IsUnknown() {
		diags = exec.Command.ElementsAs(ctx, &execConfig.Command, false)
		if diags.HasError() {
			return InstanceExecConfig{}, diags
		}
	}

	if !exec.Environment.IsNull() && !exec.Environment.IsUnknown() {
		diags = exec.Environment.ElementsAs(ctx, &execConfig.Environment, false)
		if diags.HasError() {
			return InstanceExecConfig{}, diags
		}
	}

	if !exec.WorkingDir.IsNull() && !exec.WorkingDir.IsUnknown() {
		execConfig.WorkingDir = exec.WorkingDir.ValueString()
	}

	if !exec.UserID.IsNull() && !exec.UserID.IsUnknown() {
		execConfig.UserID = exec.UserID.ValueInt64()
		execConfig.HasUserID = true
	}

	if !exec.GroupID.IsNull() && !exec.GroupID.IsUnknown() {
		execConfig.GroupID = exec.GroupID.ValueInt64()
		execConfig.HasGroupID = true
	}

	if !exec.Timeout.IsNull() && !exec.Timeout.IsUnknown() {
		timeout := exec.Timeout.ValueString()
		if timeout != "" {
			duration, err := time.ParseDuration(timeout)
			if err != nil {
				diags.AddError("Invalid timeout", err.Error())
				return InstanceExecConfig{}, diags
			}

			execConfig.Timeout = duration
			execConfig.HasTimeout = true
		}
	}

	if !exec.Trigger.IsNull() && !exec.Trigger.IsUnknown() {
		trigger := exec.Trigger.ValueString()
		if trigger != "" {
			execConfig.Trigger = trigger
		}
	}

	return execConfig, diags
}

func ExecConfigEqual(a InstanceExecConfig, b InstanceExecConfig) bool {
	if a.WorkingDir != b.WorkingDir {
		return false
	}

	if a.HasUserID != b.HasUserID || a.UserID != b.UserID {
		return false
	}

	if a.HasGroupID != b.HasGroupID || a.GroupID != b.GroupID {
		return false
	}

	if a.HasTimeout != b.HasTimeout || a.Timeout != b.Timeout {
		return false
	}

	if a.Trigger != b.Trigger {
		return false
	}

	if len(a.Command) != len(b.Command) {
		return false
	}

	for i := range a.Command {
		if a.Command[i] != b.Command[i] {
			return false
		}
	}

	if len(a.Environment) != len(b.Environment) {
		return false
	}

	for key, value := range a.Environment {
		if b.Environment[key] != value {
			return false
		}
	}

	return true
}

func RunInstanceExec(ctx context.Context, server incus.InstanceServer, instanceName string, execConfig InstanceExecConfig) (string, string, error) {
	execReq := api.InstanceExecPost{
		Command:     execConfig.Command,
		WaitForWS:   true,
		Interactive: false,
	}

	if len(execConfig.Environment) > 0 {
		execReq.Environment = execConfig.Environment
	}

	if execConfig.WorkingDir != "" {
		execReq.Cwd = execConfig.WorkingDir
	}

	if execConfig.HasUserID {
		execReq.User = uint32(execConfig.UserID)
	}

	if execConfig.HasGroupID {
		execReq.Group = uint32(execConfig.GroupID)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	execArgs := incus.InstanceExecArgs{
		Stdout:   stdout,
		Stderr:   stderr,
		DataDone: make(chan bool),
	}

	op, err := server.ExecInstance(instanceName, execReq, &execArgs)
	if err != nil {
		return stdout.String(), stderr.String(), err
	}

	waitCtx := ctx
	if execConfig.HasTimeout {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, execConfig.Timeout)
		defer cancel()
	}

	err = op.WaitContext(waitCtx)
	opAPI := op.Get()
	exitStatus := 0
	if opAPI.Metadata != nil {
		exitStatusRaw, ok := opAPI.Metadata["return"].(float64)
		if ok {
			exitStatus = int(exitStatusRaw)
		}
	}

	if err != nil {
		return stdout.String(), stderr.String(), err
	}

	if execArgs.DataDone != nil {
		select {
		case <-execArgs.DataDone:
		case <-waitCtx.Done():
			return stdout.String(), stderr.String(), waitCtx.Err()
		}
	}

	if exitStatus != 0 {
		return stdout.String(), stderr.String(), fmt.Errorf("exec returned non-zero status %d", exitStatus)
	}

	return stdout.String(), stderr.String(), nil
}

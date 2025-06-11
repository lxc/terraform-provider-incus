package common

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SetDefaultStringIfAllUndefined(defaultValue types.String, expression ...path.Expression) planmodifier.String {
	return setDefaultStringIfAllUndefinedModifier{
		defaultValue: defaultValue,
		expressions:  expression,
	}
}

func SetDefaultMapIfAllUndefined(defaultValue types.Map, expression ...path.Expression) planmodifier.Map {
	return setDefaultMapIfAllUndefinedModifier{
		defaultValue: defaultValue,
		expressions:  expression,
	}
}

type setDefaultStringIfAllUndefinedModifier struct {
	defaultValue types.String
	expressions  path.Expressions
}

func (m setDefaultStringIfAllUndefinedModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("Sets the default value %q only if none the following are set: %q", m.defaultValue.String(), m.expressions)
}

func (m setDefaultStringIfAllUndefinedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m setDefaultStringIfAllUndefinedModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}

	expressions := req.PathExpression.MergeExpressions(m.expressions...)
	allUndefinedReq := allUndefinedRequest{
		expressions: expressions,
		config:      req.Config,
	}
	allUndefinedResp := &allUndefinedResponse{}

	allUndefined(ctx, allUndefinedReq, allUndefinedResp)

	resp.Diagnostics.Append(allUndefinedResp.diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	if allUndefinedResp.allUndefined {
		resp.PlanValue = m.defaultValue
	}
}

type setDefaultMapIfAllUndefinedModifier struct {
	defaultValue types.Map
	expressions  path.Expressions
}

func (m setDefaultMapIfAllUndefinedModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("Sets the default value %q only if none the following are set: %q", m.defaultValue.String(), m.expressions)
}

func (m setDefaultMapIfAllUndefinedModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m setDefaultMapIfAllUndefinedModifier) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}

	expressions := req.PathExpression.MergeExpressions(m.expressions...)
	allUndefinedReq := allUndefinedRequest{
		expressions: expressions,
		config:      req.Config,
	}
	allUndefinedResp := &allUndefinedResponse{}

	allUndefined(ctx, allUndefinedReq, allUndefinedResp)

	resp.Diagnostics.Append(allUndefinedResp.diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	if allUndefinedResp.allUndefined {
		resp.PlanValue = m.defaultValue
	}
}

type allUndefinedRequest struct {
	expressions path.Expressions
	config      tfsdk.Config
}

type allUndefinedResponse struct {
	diagnostics  diag.Diagnostics
	allUndefined bool
}

func allUndefined(ctx context.Context, req allUndefinedRequest, resp *allUndefinedResponse) {
	resp.allUndefined = true
	for _, expression := range req.expressions {
		matchedPaths, diags := req.config.PathMatches(ctx, expression)

		resp.diagnostics.Append(diags...)

		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			var mpVal attr.Value
			diags := req.config.GetAttribute(ctx, mp, &mpVal)
			resp.diagnostics.Append(diags...)

			if diags.HasError() {
				continue
			}

			if mpVal.IsUnknown() {
				continue
			}

			if !mpVal.IsNull() {
				resp.allUndefined = false
			}
		}
	}
}

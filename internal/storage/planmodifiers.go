package storage

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type contentTypeDefault struct {
	DefaultValue    types.String
	PathExpressions path.Expressions
}

func (m contentTypeDefault) Description(ctx context.Context) string {
	return fmt.Sprintf("This attribute defaults to %q if none of %s are specified", m.DefaultValue.String(), m.PathExpressions.String())
}

func (m contentTypeDefault) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m contentTypeDefault) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, res *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
		return
	}

	expressions := req.PathExpression.MergeExpressions(m.PathExpressions...)

	isAnyConfigured := false
	for _, expression := range expressions {
		matchedPaths, diags := req.Config.PathMatches(ctx, expression)

		res.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		for _, mp := range matchedPaths {
			var mpVal attr.Value
			diags := req.Config.GetAttribute(ctx, mp, &mpVal)
			res.Diagnostics.Append(diags...)

			// Collect all errors
			if diags.HasError() {
				continue
			}

			if mpVal.IsUnknown() {
				continue
			}

			if !mpVal.IsNull() {
				isAnyConfigured = true
			}
		}
	}

	if !isAnyConfigured {
		res.PlanValue = m.DefaultValue
	}
}

func contentTypeDefaultIfUndefined(defaultValue types.String, expressions ...path.Expression) planmodifier.String {
	return contentTypeDefault{defaultValue, expressions}
}

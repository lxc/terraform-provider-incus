package common

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/lxc/incus/v6/shared/osarch"
)

type ArchitectureValidator struct{}

func (v ArchitectureValidator) Description(ctx context.Context) string {
	supportedArchitecturesList := strings.Join(osarch.SupportedArchitectures(), ", ")
	return fmt.Sprintf("Attribute architecture value must be one of: %s.", supportedArchitecturesList)
}

func (v ArchitectureValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ArchitectureValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	value := req.ConfigValue.ValueString()
	if value == "" {
		return
	}

	for _, supportedArchitecture := range osarch.SupportedArchitectures() {
		if value == supportedArchitecture {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(req.Path, "Invalid architecture",
		v.Description(ctx),
	)
}

package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Check that the protocol attr is set to a specific value.
func CheckProtocol(protocol string) validator.String {
	return checkProtocolValidator{
		protocol: protocol,
	}
}

var _ validator.String = checkProtocolValidator{}

type checkProtocolValidator struct {
	protocol string
}

func (protocolValidator checkProtocolValidator) Description(_ context.Context) string {
	return fmt.Sprintf("protocol must be set to %s", protocolValidator.protocol)
}

func (protocolValidator checkProtocolValidator) MarkdownDescription(ctx context.Context) string {
	return protocolValidator.Description(ctx)
}

func (protocolValidator checkProtocolValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	protocolPath := req.Path.ParentPath().AtName("protocol")

	var protocol types.String

	diags := req.Config.GetAttribute(ctx, protocolPath, &protocol)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if protocol.IsNull() || protocol.IsUnknown() {
		return
	}

	if protocol.ValueString() != protocolValidator.protocol {
		resp.Diagnostics.AddAttributeError(
			protocolPath,
			"Protocol value invalid",
			protocolValidator.Description(ctx),
		)
	}
}

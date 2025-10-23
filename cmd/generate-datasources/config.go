package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config contains a map of the specification for all entities (resources),
// for which the code for the data source should be generated.
// The key of the map is the `name` of the resource, e.g. `network`.
// The key MUST be singular and in snake case, e.g. `network_forward`.
type Config map[string]*Entity

type Entity struct {
	// Additional description about the resource. This is added to the
	// introduction section of resource in the documentation.
	Description string `yaml:"description"`

	// Content for the optinal notes section at the end of the data source
	// documention.
	Notes string `yaml:"notes"`

	// Name of the package, this data source belongs to.
	// This also defines the path of the package in `internal/`.
	PackageName string `yaml:"package-name"`

	// Name property of the resource. If not set, this defaults to `name`.
	// But some entities do not have a `name` property and therefore an other
	// property is name defining.
	ObjectNamePropertyName string `yaml:"object-name-property-name"`

	// Default value used in the documentation for the name defining property.
	// Defaults to: `default`.
	ObjectNamePropertyDefaultValue string `yaml:"object-name-property-default-value"`

	// Method of the Incus client to get the resource, e.g. `GetNetwork`.
	//
	// E.g. from https://github.com/lxc/incus/blob/3da8fcd06c4f7ee3cb9388127e6071244db7ac8f/client/incus_networks.go#L104
	IncusGetMethod string `yaml:"incus-get-method"`

	// Name of the parent entity, if any.
	// Resources like network forwards have a parent, in this case a network.
	// If this is the case, the name of the parent needs to be specidied.
	ParentName string `yaml:"parent"`

	// If a resource has no project attribute.
	// Most resources do have a project attribute. If this is not the case,
	// `has-no-project` needs to be set to `true`.
	HasNoProject bool `yaml:"has-no-project"`

	// If a resource has a target.
	// Most resources do not have a target attribute. If a resource does have a
	// target, `has-target` needs to be set to `true`.
	HasTarget bool `yaml:"has-target"`

	// If a resource has no status attribute.
	// Most resources do have a status attribute. If this is not the case,
	// `has-no-status` needs to be set to `true`.
	HasNoStatus bool `yaml:"has-no-status"`

	// If a resource has a location, mutual exclusive with has-locations.
	// Some resources are location aware. If this is the case for a resource,
	// `has-location` needs to be set to `true`.
	HasLocation bool `yaml:"has-location"`

	// If a resource has multiple locations, mutual exclusive with has-location.
	// Some resources can be assigned to multiple locations. If this is the case
	// for a resource, `has-locations` needs to be set to `true`.
	HasLocations bool `yaml:"has-locations"`

	// Name in snake case of an extra ID attribute, which is specific to the
	// respective resource type.
	// As of now, the only resource requiring an extra ID defining attribute is
	// storage volume, where the `type` is also part of the ID.
	ExtraIDAttribute ExtraAttribute `yaml:"extra-id-attribute"`

	// List of extra attributes, which are specific to the respective resource type.
	//
	// See definition of type ExtraAttribute for details.
	//
	// The following attributes are handled automatically be the code generator
	// and must therefore not be listed in `extra-attributes`:
	//
	//   * `name` (or if the resource does not have a `name` attribute, the attribute referenced in `object-name-property-name`)
	//   * The attributes mentioned in `extra-id-attribute`, if any
	//   * `parent` (if `parent` is not empty)
	//   * `project`
	//   * `target`
	//   * `remote`
	//   * `description`
	//   * `config`
	//   * `status` (if `has-no-status` is not set to `true`)
	//   * `location` (if `has-location` is set to `true`)
	//   * `locations` (if `has-locations` is set to `true`)
	ExtraAttributes []ExtraAttribute `yaml:"extra-attributes"`

	// Map of additional description added to the documentation for the
	// automatically handled attributes (see list above).
	// Is added to the documentation "as-is", may contain markdown.
	ExtraDescriptions map[string]string `yaml:"extra-descriptions"`
}

// ExtraAttribute contains the specification for an attribute of a resource.
type ExtraAttribute struct {
	// Name of the attribute in snake case, e.g. `type`.
	Name string `yaml:"name"`

	// Data type of the attribute, e.g. `string`.
	// Supported types are (mapping directly to the corresponding types from Terraform):
	//
	//   * `bool`
	//   * `list`
	//   * `map`
	//   * `object`
	//   * `string`
	//
	// Additionally, the following special types are supported:
	// (the internal types are recognizable by the leading `_`)
	//
	//   * `_device`: represents devices as used in instances and profiles. On the
	//                API, these are represented as map[string]map[string]string,
	//                which is mapped to a set of blocks of device information
	//                consisting of name, type and properties.
	//                This is a special type with the names and the logic hard
	//                coded. With this, it can only be used to represent devices.
	Type string `yaml:"type"`

	// If the `type` is `list` or `map`, `element-type` defines the Terraform type
	// of the elements contained in the list or map.
	// Supported types for list are:
	//
	//   * `bool`
	//   * `list`
	//   * `map`
	//   * `object`
	//   * `string`
	//
	// Supported types for `map`:
	//
	//   * `bool`
	//   * `list`
	//   * `object`
	//   * `string`
	//
	// Be aware, that nesting of map in map is not supported by Terraform.
	ElementType *ExtraAttribute `yaml:"element-type"`

	// If the `type` or the `element-type` is `object`, the type of the attribtues of
	// the object need to be defined. This is a list, listing each possible attribute of
	// the object.
	AttrTypes []*ExtraAttribute `yaml:"attr-types"`

	// Description of the extra attribute. This is added to the documentation.
	Description string `yaml:"description"`
}

func (c *Config) LoadConfig(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(contents, c)
}

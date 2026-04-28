package network

import (
	"reflect"
	"testing"
)

func TestNetworkConfig_nodeSpecificKeys(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{name: "parent", key: "parent", expected: true},
		{name: "bridge external interfaces", key: "bridge.external_interfaces", expected: true},
		{name: "bgp ipv4 nexthop", key: "bgp.ipv4.nexthop", expected: true},
		{name: "bgp ipv6 nexthop", key: "bgp.ipv6.nexthop", expected: true},
		{name: "tunnel interface", key: "tunnel.foo.interface", expected: true},
		{name: "tunnel local", key: "tunnel.foo.local", expected: true},
		{name: "vlan", key: "vlan", expected: false},
		{name: "ipv4 address", key: "ipv4.address", expected: false},
		{name: "tunnel remote", key: "tunnel.foo.remote", expected: false},
		{name: "nested tunnel interface", key: "tunnel.foo.bar.interface", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := isNodeSpecificNetworkConfig(tt.key)
			if actual != tt.expected {
				t.Fatalf("isNodeSpecificNetworkConfig(%q) = %t, want %t", tt.key, actual, tt.expected)
			}
		})
	}
}

func TestNetworkConfig_stripClusterWideKeys(t *testing.T) {
	config := map[string]string{
		"parent":                     "bond0",
		"bridge.external_interfaces": "eth0",
		"bgp.ipv4.nexthop":           "192.0.2.1",
		"bgp.ipv6.nexthop":           "2001:db8::1",
		"tunnel.foo.interface":       "eth1",
		"tunnel.foo.local":           "192.0.2.2",
		"vlan":                       "400",
		"ipv4.address":               "10.150.19.1/24",
		"tunnel.foo.remote":          "192.0.2.3",
		"tunnel.foo.bar.interface":   "eth2",
	}

	actual := stripClusterWideNetworkConfig(config)
	expected := map[string]string{
		"parent":                     "bond0",
		"bridge.external_interfaces": "eth0",
		"bgp.ipv4.nexthop":           "192.0.2.1",
		"bgp.ipv6.nexthop":           "2001:db8::1",
		"tunnel.foo.interface":       "eth1",
		"tunnel.foo.local":           "192.0.2.2",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("stripClusterWideNetworkConfig() = %#v, want %#v", actual, expected)
	}
}

package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWords(t *testing.T) {
	tests := []struct {
		in string

		wantCamelCase  string
		wantPascalCase string
		wantKebabCase  string
		wantTitleCase  string
		wantWords      string
	}{
		{
			in: "foo",

			wantCamelCase:  "foo",
			wantPascalCase: "Foo",
			wantKebabCase:  "foo",
			wantTitleCase:  "Foo",
			wantWords:      "foo",
		},
		{
			in: "foo_bar",

			wantCamelCase:  "fooBar",
			wantPascalCase: "FooBar",
			wantKebabCase:  "foo-bar",
			wantTitleCase:  "Foo Bar",
			wantWords:      "foo bar",
		},
		{
			in: "acl",

			wantCamelCase:  "acl",
			wantPascalCase: "ACL",
			wantKebabCase:  "acl",
			wantTitleCase:  "ACL",
			wantWords:      "ACL",
		},
		{
			in: "acls",

			wantCamelCase:  "acls",
			wantPascalCase: "ACLs",
			wantKebabCase:  "acls",
			wantTitleCase:  "ACLs",
			wantWords:      "ACLs",
		},
		{
			in: "foo_acl",

			wantCamelCase:  "fooACL",
			wantPascalCase: "FooACL",
			wantKebabCase:  "foo-acl",
			wantTitleCase:  "Foo ACL",
			wantWords:      "foo ACL",
		},
		{
			in: "foo_acls",

			wantCamelCase:  "fooACLs",
			wantPascalCase: "FooACLs",
			wantKebabCase:  "foo-acls",
			wantTitleCase:  "Foo ACLs",
			wantWords:      "foo ACLs",
		},
		{
			in: "acl_foo",

			wantCamelCase:  "aclFoo",
			wantPascalCase: "ACLFoo",
			wantKebabCase:  "acl-foo",
			wantTitleCase:  "ACL Foo",
			wantWords:      "ACL foo",
		},
	}

	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.wantCamelCase, CamelCase(tc.in))
			require.Equal(t, tc.wantPascalCase, PascalCase(tc.in))
			require.Equal(t, tc.wantKebabCase, KebabCase(tc.in))
			require.Equal(t, tc.wantTitleCase, TitleCase(tc.in))
			require.Equal(t, tc.wantWords, Words(tc.in))
		})
	}
}

package authz

import _ "embed"

type PolicyDocument struct {
	Name string
	Text string
}

//go:embed base.cedar
var basePolicies string

/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package auth

import (
	"github.com/pkg/errors"
	networking "k8s.io/api/networking/v1"

	"github.com/stolostron/management-ingress/pkg/ingress"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/parser"
	"github.com/stolostron/management-ingress/pkg/ingress/resolver"
)

type at struct {
	r resolver.Resolver
}

// NewParser creates a new secure upstream annotation parser
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return at{r}
}

// Parse parses the annotations contained in the ingress
// rule used to indicate if the upstream servers should use SSL
func (a at) Parse(ing *networking.Ingress) (interface{}, error) {
	ca, _ := parser.GetStringAnnotation("auth-type", ing)
	if ca != ingress.IDToken && ca != ingress.AccessToken {
		return "", errors.Errorf("Auth type %v is not supported", ca)
	}
	return ca, nil
}

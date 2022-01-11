/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rewrite

import (
	extensions "k8s.io/api/extensions/v1beta1"

	"github.com/stolostron/management-ingress/pkg/ingress/annotations/parser"
	"github.com/stolostron/management-ingress/pkg/ingress/resolver"
)

// Config describes the per location redirect config
type Config struct {
	// Target URI where the traffic must be redirected
	Target string `json:"target"`
	// AddBaseURL indicates if is required to add a base tag in the head
	// of the responses from the upstream servers
	AddBaseURL bool `json:"addBaseUrl"`
	// BaseURLScheme override for the scheme passed to the base tag
	BaseURLScheme string `json:"baseUrlScheme"`
	// AppRoot defines the Application Root that the Controller must redirect if it's in '/' context
	AppRoot string `json:"appRoot"`
}

// Equal tests for equality between two Redirect types
func (r1 *Config) Equal(r2 *Config) bool {
	if r1 == r2 {
		return true
	}
	if r1 == nil || r2 == nil {
		return false
	}
	if r1.Target != r2.Target {
		return false
	}
	if r1.AddBaseURL != r2.AddBaseURL {
		return false
	}
	if r1.BaseURLScheme != r2.BaseURLScheme {
		return false
	}
	if r1.AppRoot != r2.AppRoot {
		return false
	}

	return true
}

type rewrite struct {
	r resolver.Resolver
}

// NewParser creates a new reqrite annotation parser
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return rewrite{r}
}

// ParseAnnotations parses the annotations contained in the ingress
// rule used to rewrite the defined paths
func (a rewrite) Parse(ing *extensions.Ingress) (interface{}, error) {
	rt, _ := parser.GetStringAnnotation("rewrite-target", ing)
	abu, _ := parser.GetBoolAnnotation("add-base-url", ing)
	bus, _ := parser.GetStringAnnotation("base-url-scheme", ing)
	ar, _ := parser.GetStringAnnotation("app-root", ing)

	return &Config{
		Target:        rt,
		AddBaseURL:    abu,
		BaseURLScheme: bus,
		AppRoot:       ar,
	}, nil
}

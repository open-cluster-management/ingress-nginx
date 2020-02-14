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

package proxy

import (
	extensions "k8s.io/api/extensions/v1beta1"

	"github.com/open-cluster-management/management-ingress/pkg/ingress/annotations/parser"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/resolver"
)

var DefaultProxyConfig = Config{
	BodySize:       "1m",
	ConnectTimeout: 5,
	SendTimeout:    60,
	ReadTimeout:    60,
	BufferSize:     "4k",
}

// Config returns the proxy timeout to use in the upstream server/s
type Config struct {
	BodySize       string `json:"bodySize"`
	ConnectTimeout int    `json:"connectTimeout"`
	SendTimeout    int    `json:"sendTimeout"`
	ReadTimeout    int    `json:"readTimeout"`
	BufferSize     string `json:"bufferSize"`
}

// Equal tests for equality between two Configuration types
func (l1 *Config) Equal(l2 *Config) bool {
	if l1 == l2 {
		return true
	}
	if l1 == nil || l2 == nil {
		return false
	}
	if l1.BodySize != l2.BodySize {
		return false
	}
	if l1.ConnectTimeout != l2.ConnectTimeout {
		return false
	}
	if l1.SendTimeout != l2.SendTimeout {
		return false
	}
	if l1.ReadTimeout != l2.ReadTimeout {
		return false
	}
	if l1.BufferSize != l2.BufferSize {
		return false
	}

	return true
}

type proxy struct {
	r resolver.Resolver
}

// NewParser creates a new reverse proxy configuration annotation parser
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return proxy{r}
}

// ParseAnnotations parses the annotations contained in the ingress
// rule used to configure upstream check parameters
func (a proxy) Parse(ing *extensions.Ingress) (interface{}, error) {
	ct, err := parser.GetIntAnnotation("proxy-connect-timeout", ing)
	if err != nil {
		ct = DefaultProxyConfig.ConnectTimeout
	}

	st, err := parser.GetIntAnnotation("proxy-send-timeout", ing)
	if err != nil {
		st = DefaultProxyConfig.SendTimeout
	}

	rt, err := parser.GetIntAnnotation("proxy-read-timeout", ing)
	if err != nil {
		rt = DefaultProxyConfig.ReadTimeout
	}

	bufs, err := parser.GetStringAnnotation("proxy-buffer-size", ing)
	if err != nil || bufs == "" {
		bufs = DefaultProxyConfig.BufferSize
	}

	bs, err := parser.GetStringAnnotation("proxy-body-size", ing)
	if err != nil || bs == "" {
		bs = DefaultProxyConfig.BodySize
	}

	return &Config{bs, ct, st, rt, bufs}, nil
}

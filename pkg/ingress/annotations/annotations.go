/*
Copyright 2017 The Kubernetes Authors.

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

package annotations

import (
	"github.com/golang/glog"
	"github.com/imdario/mergo"

	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stolostron/management-ingress/pkg/ingress/annotations/auth"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/authz"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/connection"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/locationmodifier"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/parser"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/proxy"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/rewrite"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/secureupstream"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/snippet"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/upstreamhashby"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/upstreamuri"
	"github.com/stolostron/management-ingress/pkg/ingress/annotations/xforwardedprefix"
	"github.com/stolostron/management-ingress/pkg/ingress/errors"
	"github.com/stolostron/management-ingress/pkg/ingress/resolver"
)

// DeniedKeyName name of the key that contains the reason to deny a location
const DeniedKeyName = "Denied"

// Ingress defines the valid annotations present in one NGINX Ingress rule
type Ingress struct {
	metav1.ObjectMeta
	AuthType             string
	AuthzType            string
	ConfigurationSnippet string
	LocationModifier     string
	UpstreamHashBy       string
	UpstreamURI          string
	Rewrite              rewrite.Config
	SecureUpstream       secureupstream.Config
	XForwardedPrefix     bool
	Proxy                proxy.Config
	Connection           connection.Config
}

// Extractor defines the annotation parsers to be used in the extraction of annotations
type Extractor struct {
	annotations map[string]parser.IngressAnnotation
}

// NewAnnotationExtractor creates a new annotations extractor
func NewAnnotationExtractor(cfg resolver.Resolver) Extractor {
	return Extractor{
		map[string]parser.IngressAnnotation{
			"AuthType":             auth.NewParser(cfg),
			"AuthzType":            authz.NewParser(cfg),
			"ConfigurationSnippet": snippet.NewParser(cfg),
			"SecureUpstream":       secureupstream.NewParser(cfg),
			"Rewrite":              rewrite.NewParser(cfg),
			"UpstreamHashBy":       upstreamhashby.NewParser(cfg),
			"XForwardedPrefix":     xforwardedprefix.NewParser(cfg),
			"LocationModifier":     locationmodifier.NewParser(cfg),
			"UpstreamURI":          upstreamuri.NewParser(cfg),
			"Proxy":                proxy.NewParser(cfg),
			"Connection":           connection.NewParser(cfg),
		},
	}
}

// Extract extracts the annotations from an Ingress
func (e Extractor) Extract(ing *extensions.Ingress) *Ingress {
	pia := &Ingress{
		ObjectMeta: ing.ObjectMeta,
	}

	data := make(map[string]interface{})
	for name, annotationParser := range e.annotations {
		val, err := annotationParser.Parse(ing)
		glog.V(5).Infof("annotation %v in Ingress %v/%v: %v", name, ing.GetNamespace(), ing.GetName(), val)
		if err != nil {
			if errors.IsMissingAnnotations(err) {
				continue
			}

			if !errors.IsLocationDenied(err) {
				continue
			}

			_, alreadyDenied := data[DeniedKeyName]
			if !alreadyDenied {
				data[DeniedKeyName] = err
				glog.Errorf("error reading %v annotation in Ingress %v/%v: %v", name, ing.GetNamespace(), ing.GetName(), err)
				continue
			}

			glog.V(5).Infof("error reading %v annotation in Ingress %v/%v: %v", name, ing.GetNamespace(), ing.GetName(), err)
		}

		if val != nil {
			data[name] = val
		}
	}

	err := mergo.MapWithOverwrite(pia, data)
	if err != nil {
		glog.Errorf("unexpected error merging extracted annotations: %v", err)
	}

	return pia
}

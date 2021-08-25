/*
Licensed Materials - Property of IBM
cfc
@ Copyright IBM Corp. 2018 All Rights Reserved
US Government Users Restricted Rights - Use, duplication or disclosure
restricted by GSA ADP Schedule Contract with IBM Corp.
*/

// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package secureupstream

import (
	"fmt"
	"testing"

	api "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/open-cluster-management/management-ingress/pkg/ingress/annotations/parser"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/resolver"
)

func buildIngress() *networking.Ingress {
	defaultBackend := networking.IngressBackend{
		Service: &networking.IngressServiceBackend{
			Name: "default-backend",
			Port: networking.ServiceBackendPort{
				Number: 80,
			},
		},
	}

	return &networking.Ingress{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "foo",
			Namespace: api.NamespaceDefault,
		},
		Spec: networking.IngressSpec{
			DefaultBackend: &networking.IngressBackend{
				Service: &networking.IngressServiceBackend{
					Name: "default-backend",
					Port: networking.ServiceBackendPort{
						Number: 80,
					},
				},
			},
			Rules: []networking.IngressRule{
				{
					Host: "foo.bar.com",
					IngressRuleValue: networking.IngressRuleValue{
						HTTP: &networking.HTTPIngressRuleValue{
							Paths: []networking.HTTPIngressPath{
								{
									Path:    "/foo",
									Backend: defaultBackend,
								},
							},
						},
					},
				},
			},
		},
	}
}

type mockCfg struct {
	resolver.Mock
	certs map[string]resolver.AuthSSLCert
}

func (cfg mockCfg) GetAuthCertificate(secret string) (*resolver.AuthSSLCert, error) {
	if cert, ok := cfg.certs[secret]; ok {
		return &cert, nil
	}
	return nil, fmt.Errorf("secret not found: %v", secret)
}

func TestAnnotations(t *testing.T) {
	ing := buildIngress()
	data := map[string]string{}
	data[parser.GetAnnotationWithPrefix("secure-backends")] = "true"
	data[parser.GetAnnotationWithPrefix("secure-verify-ca-secret")] = "secure-verify-ca"
	ing.SetAnnotations(data)

	_, err := NewParser(mockCfg{
		certs: map[string]resolver.AuthSSLCert{
			"default/secure-verify-ca": {},
		},
	}).Parse(ing)
	if err != nil {
		t.Errorf("Unexpected error on ingress: %v", err)
	}
}

func TestSecretNotFound(t *testing.T) {
	ing := buildIngress()
	data := map[string]string{}
	data[parser.GetAnnotationWithPrefix("secure-backends")] = "true"
	data[parser.GetAnnotationWithPrefix("secure-verify-ca-secret")] = "secure-verify-ca"
	ing.SetAnnotations(data)
	_, err := NewParser(mockCfg{}).Parse(ing)
	if err == nil {
		t.Error("Expected secret not found error on ingress")
	}
}

func TestSecretOnNonSecure(t *testing.T) {
	ing := buildIngress()
	data := map[string]string{}
	data[parser.GetAnnotationWithPrefix("secure-backends")] = "false"
	data[parser.GetAnnotationWithPrefix("secure-verify-ca-secret")] = "secure-verify-ca"
	ing.SetAnnotations(data)
	_, err := NewParser(mockCfg{
		certs: map[string]resolver.AuthSSLCert{
			"default/secure-verify-ca": {},
		},
	}).Parse(ing)
	if err == nil {
		t.Error("Expected CA secret on non secure backend error on ingress")
	}
}

func TestClientSecretNotFound(t *testing.T) {
	ing := buildIngress()
	data := map[string]string{}
	data[parser.GetAnnotationWithPrefix("secure-backends")] = "true"
	data[parser.GetAnnotationWithPrefix("secure-client-ca-secret")] = "secure-client-ca"
	ing.SetAnnotations(data)
	_, err := NewParser(mockCfg{}).Parse(ing)
	if err == nil {
		t.Error("Expected client secret not found error on ingress")
	}
}

func TestClientSecretOnNonSecure(t *testing.T) {
	ing := buildIngress()
	data := map[string]string{}
	data[parser.GetAnnotationWithPrefix("secure-backends")] = "false"
	data[parser.GetAnnotationWithPrefix("secure-client-ca-secret")] = "secure-client-ca"
	ing.SetAnnotations(data)
	_, err := NewParser(mockCfg{
		certs: map[string]resolver.AuthSSLCert{
			"default/secure-client-ca": {},
		},
	}).Parse(ing)
	if err == nil {
		t.Error("Expected Client CA secret on non secure backend error on ingress")
	}
}

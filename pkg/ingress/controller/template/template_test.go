/*
Copyright 2015 The Kubernetes Authors.

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

package template

import (
	"net"
	"strings"
	"testing"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/annotations/rewrite"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/resolver"
)

var (
	// TODO: add tests for secure endpoints
	tmplFuncTestcases = map[string]struct {
		Path             string
		Target           string
		Location         string
		ProxyPass        string
		AddBaseURL       bool
		BaseURLScheme    string
		Sticky           bool
		XForwardedPrefix bool
	}{
		"invalid redirect / to /": {"/", "/", "/", "proxy_pass http://upstream-name;", false, "", false, false},
		"redirect / to /jenkins": {"/", "/jenkins", "~* /",
			`
	    rewrite /(.*) /jenkins/$1 break;
	    proxy_pass http://upstream-name;
	    `, false, "", false, false},
		"redirect /something to /": {"/something", "/", `~* ^/something\/?(?<baseuri>.*)`, `
	    rewrite /something/(.*) /$1 break;
	    rewrite /something / break;
	    proxy_pass http://upstream-name;
	    `, false, "", false, false},
		"redirect /end-with-slash/ to /not-root": {"/end-with-slash/", "/not-root", "~* ^/end-with-slash/(?<baseuri>.*)", `
	    rewrite /end-with-slash/(.*) /not-root/$1 break;
	    proxy_pass http://upstream-name;
	    `, false, "", false, false},
		"redirect /something-complex to /not-root": {"/something-complex", "/not-root", `~* ^/something-complex\/?(?<baseuri>.*)`, `
	    rewrite /something-complex/(.*) /not-root/$1 break;
	    proxy_pass http://upstream-name;
	    `, false, "", false, false},
		"redirect / to /jenkins and rewrite": {"/", "/jenkins", "~* /", `
	    rewrite /(.*) /jenkins/$1 break;
	    proxy_pass http://upstream-name;
	    subs_filter '(<(?:H|h)(?:E|e)(?:A|a)(?:D|d)(?:[^">]|"[^"]*")*>)' '$1<base href="$scheme://$http_host/$baseuri">' ro;
	    `, true, "", false, false},
		"redirect /something to / and rewrite": {"/something", "/", `~* ^/something\/?(?<baseuri>.*)`, `
	    rewrite /something/(.*) /$1 break;
	    rewrite /something / break;
	    proxy_pass http://upstream-name;
	    subs_filter '(<(?:H|h)(?:E|e)(?:A|a)(?:D|d)(?:[^">]|"[^"]*")*>)' '$1<base href="$scheme://$http_host/something/$baseuri">' ro;
	    `, true, "", false, false},
		"redirect /end-with-slash/ to /not-root and rewrite": {"/end-with-slash/", "/not-root", `~* ^/end-with-slash/(?<baseuri>.*)`, `
	    rewrite /end-with-slash/(.*) /not-root/$1 break;
	    proxy_pass http://upstream-name;
	    subs_filter '(<(?:H|h)(?:E|e)(?:A|a)(?:D|d)(?:[^">]|"[^"]*")*>)' '$1<base href="$scheme://$http_host/end-with-slash/$baseuri">' ro;
	    `, true, "", false, false},
		"redirect /something-complex to /not-root and rewrite": {"/something-complex", "/not-root", `~* ^/something-complex\/?(?<baseuri>.*)`, `
	    rewrite /something-complex/(.*) /not-root/$1 break;
	    proxy_pass http://upstream-name;
	    subs_filter '(<(?:H|h)(?:E|e)(?:A|a)(?:D|d)(?:[^">]|"[^"]*")*>)' '$1<base href="$scheme://$http_host/something-complex/$baseuri">' ro;
	    `, true, "", false, false},
		"redirect /something to / and rewrite with specific scheme": {"/something", "/", `~* ^/something\/?(?<baseuri>.*)`, `
	    rewrite /something/(.*) /$1 break;
	    rewrite /something / break;
	    proxy_pass http://upstream-name;
	    subs_filter '(<(?:H|h)(?:E|e)(?:A|a)(?:D|d)(?:[^">]|"[^"]*")*>)' '$1<base href="http://$http_host/something/$baseuri">' ro;
	    `, true, "http", false, false},
		"redirect / to /something with sticky enabled": {"/", "/something", `~* /`, `
	    rewrite /(.*) /something/$1 break;
	    proxy_pass http://upstream-name;
	    `, false, "http", true, false},
		"add the X-Forwarded-Prefix header": {"/there", "/something", `~* ^/there\/?(?<baseuri>.*)`, `
	    rewrite /there/(.*) /something/$1 break;
	    proxy_set_header X-Forwarded-Prefix "/there/";
	    proxy_pass http://upstream-name;
	    `, false, "http", true, true},
	}
)

func TestFormatIP(t *testing.T) {
	cases := map[string]struct {
		Input, Output string
	}{
		"ipv4-localhost": {"127.0.0.1", "127.0.0.1"},
		"ipv4-internet":  {"8.8.8.8", "8.8.8.8"},
		"ipv6-localhost": {"::1", "[::1]"},
		"ipv6-internet":  {"2001:4860:4860::8888", "[2001:4860:4860::8888]"},
		"invalid-ip":     {"nonsense", "nonsense"},
		"empty-ip":       {"", ""},
	}
	for k, tc := range cases {
		res := formatIP(tc.Input)
		if res != tc.Output {
			t.Errorf("%s: called formatIp('%s'); expected '%v' but returned '%v'", k, tc.Input, tc.Output, res)
		}
	}
}

func TestBuildLocation(t *testing.T) {
	for k, tc := range tmplFuncTestcases {
		loc := &ingress.Location{
			Path:    tc.Path,
			Rewrite: rewrite.Config{Target: tc.Target, AddBaseURL: tc.AddBaseURL},
		}

		newLoc := buildLocation(loc)
		if tc.Location != newLoc {
			t.Errorf("%s: expected '%v' but returned %v", k, tc.Location, newLoc)
		}
	}
}

func TestBuildProxyPass(t *testing.T) {
	defaultBackend := "upstream-name"
	defaultHost := "example.com"

	for k, tc := range tmplFuncTestcases {
		loc := &ingress.Location{
			Path:             tc.Path,
			Rewrite:          rewrite.Config{Target: tc.Target, AddBaseURL: tc.AddBaseURL, BaseURLScheme: tc.BaseURLScheme},
			Backend:          defaultBackend,
			XForwardedPrefix: tc.XForwardedPrefix,
		}

		backends := []*ingress.Backend{}
		if tc.Sticky {
			backends = []*ingress.Backend{
				{
					Name: defaultBackend,
				},
			}
		}

		pp := buildProxyPass(defaultHost, backends, loc)
		if !strings.EqualFold(tc.ProxyPass, pp) {
			t.Errorf("%s: expected \n'%v'\nbut returned \n'%v'", k, tc.ProxyPass, pp)
		}
	}
}

func TestBuildClientBodyBufferSize(t *testing.T) {
	a := isValidClientBodyBufferSize("1000")
	if a != true {
		t.Errorf("Expected '%v' but returned '%v'", true, a)
	}
	b := isValidClientBodyBufferSize("1000k")
	if b != true {
		t.Errorf("Expected '%v' but returned '%v'", true, b)
	}
	c := isValidClientBodyBufferSize("1000m")
	if c != true {
		t.Errorf("Expected '%v' but returned '%v'", true, c)
	}
	d := isValidClientBodyBufferSize("1000km")
	if d != false {
		t.Errorf("Expected '%v' but returned '%v'", false, d)
	}
	e := isValidClientBodyBufferSize("1000mk")
	if e != false {
		t.Errorf("Expected '%v' but returned '%v'", false, e)
	}
	f := isValidClientBodyBufferSize("1000kk")
	if f != false {
		t.Errorf("Expected '%v' but returned '%v'", false, f)
	}
	g := isValidClientBodyBufferSize("1000mm")
	if g != false {
		t.Errorf("Expected '%v' but returned '%v'", false, g)
	}
	h := isValidClientBodyBufferSize(nil)
	if h != false {
		t.Errorf("Expected '%v' but returned '%v'", false, h)
	}
	i := isValidClientBodyBufferSize("")
	if i != false {
		t.Errorf("Expected '%v' but returned '%v'", false, i)
	}
}

func TestBuildForwardedFor(t *testing.T) {
	inputStr := "X-Forwarded-For"
	outputStr := buildForwardedFor(inputStr)

	validStr := "$http_x_forwarded_for"

	if outputStr != validStr {
		t.Errorf("Expected '%v' but returned '%v'", validStr, outputStr)
	}
}

func TestBuildResolvers(t *testing.T) {
	ipOne := net.ParseIP("192.0.0.1")
	ipTwo := net.ParseIP("2001:db8:1234:0000:0000:0000:0000:0000")
	ipList := []net.IP{ipOne, ipTwo}

	validResolver := "resolver 192.0.0.1 [2001:db8:1234::] valid=30s;"
	resolver := buildResolvers(ipList)

	if resolver != validResolver {
		t.Errorf("Expected '%v' but returned '%v'", validResolver, resolver)
	}
}

func TestBuildVerifySSL(t *testing.T) {
	defaultBackend := "upstream-name"

	validBackend := "proxy_ssl_verify off;"

	loc := &ingress.Location{
		Path:    "/",
		Backend: defaultBackend,
	}

	backends := []*ingress.Backend{
		{
			Name:         defaultBackend,
			Secure:       true,
			SecureCACert: resolver.AuthSSLCert{},
		},
	}

	sslBackend := buildSSLVeify(backends, loc)
	if sslBackend != validBackend {
		t.Errorf("Expected '%v' but returned '%v'", validBackend, sslBackend)
	}
}

func TestBuildClientCAAuth(t *testing.T) {
	defaultBackend := "upstream-name"

	validBackend := `
	    proxy_ssl_certificate /test/test.crt;
	    proxy_ssl_certificate_key /test/test.crt;
	    `

	loc := &ingress.Location{
		Path:    "/",
		Backend: defaultBackend,
	}

	backends := []*ingress.Backend{
		{
			Name:   defaultBackend,
			Secure: true,
			ClientCACert: resolver.AuthSSLCert{
				Secret:      "test",
				PemFileName: "/test/test.crt",
			},
		},
	}

	sslBackend := buildClientCAAuth(backends, loc)
	if sslBackend != validBackend {
		t.Errorf("Expected '%v' but returned '%v'", validBackend, sslBackend)
	}
}

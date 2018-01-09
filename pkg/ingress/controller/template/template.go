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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	text_template "text/template"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/file"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress"
	"github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/ingress/controller/config"
	ing_net "github.ibm.com/IBMPrivateCloud/icp-management-ingress/pkg/net"
)

const (
	slash         = "/"
	nonIdempotent = "non_idempotent"
	defBufferSize = 65535
)

// Template ...
type Template struct {
	tmpl *text_template.Template
	//fw   watch.FileWatcher
	bp *BufferPool
}

//NewTemplate returns a new Template instance or an
//error if the specified template file contains errors
func NewTemplate(file string, fs file.Filesystem) (*Template, error) {
	data, err := fs.ReadFile(file)
	if err != nil {
		return nil, errors.Wrapf(err, "unexpected error reading template %v", file)
	}

	tmpl, err := text_template.New("nginx.tmpl").Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, err
	}

	return &Template{
		tmpl: tmpl,
		bp:   NewBufferPool(defBufferSize),
	}, nil
}

// Write populates a buffer using a template with NGINX configuration
// and the servers and upstreams created by Ingress rules
func (t *Template) Write(conf config.TemplateConfig) ([]byte, error) {
	tmplBuf := t.bp.Get()
	defer t.bp.Put(tmplBuf)

	outCmdBuf := t.bp.Get()
	defer t.bp.Put(outCmdBuf)

	if glog.V(3) {
		b, err := json.Marshal(conf)
		if err != nil {
			glog.Errorf("unexpected error: %v", err)
		}
		glog.Infof("NGINX configuration: %v", string(b))
	}

	err := t.tmpl.Execute(tmplBuf, conf)
	if err != nil {
		return nil, err
	}

	// squeezes multiple adjacent empty lines to be single
	// spaced this is to avoid the use of regular expressions
	cmd := exec.Command("/opt/ibm/router/clean-nginx-conf.sh")
	cmd.Stdin = tmplBuf
	cmd.Stdout = outCmdBuf
	if err := cmd.Run(); err != nil {
		glog.Warningf("unexpected error cleaning template: %v", err)
		return tmplBuf.Bytes(), nil
	}

	return outCmdBuf.Bytes(), nil
}

var (
	funcMap = text_template.FuncMap{
		"empty": func(input interface{}) bool {
			check, ok := input.(string)
			if ok {
				return len(check) == 0
			}
			return true
		},
		"buildLocation":     buildLocation,
		"buildProxyPass":    buildProxyPass,
		"buildResolvers":    buildResolvers,
		"buildUpstreamName": buildUpstreamName,
		"getenv":            os.Getenv,
		"contains":          strings.Contains,
		"hasPrefix":         strings.HasPrefix,
		"hasSuffix":         strings.HasSuffix,
		"toUpper":           strings.ToUpper,
		"toLower":           strings.ToLower,
		"buildForwardedFor": buildForwardedFor,
		"formatIP":          formatIP,
		"serverConfig": func(all config.TemplateConfig, server *ingress.Server) interface{} {
			return struct{ First, Second interface{} }{all, server}
		},
	}
)

// formatIP will wrap IPv6 addresses in [] and return IPv4 addresses
// without modification. If the input cannot be parsed as an IP address
// it is returned without modification.
func formatIP(input string) string {
	ip := net.ParseIP(input)
	if ip == nil {
		return input
	}
	if v4 := ip.To4(); v4 != nil {
		return input
	}
	return fmt.Sprintf("[%s]", input)
}

// buildResolvers returns the resolvers reading the /etc/resolv.conf file
func buildResolvers(input interface{}) string {
	// NGINX need IPV6 addresses to be surrounded by brackets
	nss, ok := input.([]net.IP)
	if !ok {
		glog.Errorf("expected a '[]net.IP' type but %T was returned", input)
		return ""
	}

	if len(nss) == 0 {
		return ""
	}

	r := []string{"resolver"}
	for _, ns := range nss {
		if ing_net.IsIPV6(ns) {
			r = append(r, fmt.Sprintf("[%v]", ns))
		} else {
			r = append(r, fmt.Sprintf("%v", ns))
		}
	}
	r = append(r, "valid=30s;")

	return strings.Join(r, " ")
}

// buildLocation produces the location string, if the ingress has redirects
// (specified through the nginx.ingress.kubernetes.io/rewrite-to annotation)
func buildLocation(input interface{}) string {
	location, ok := input.(*ingress.Location)
	if !ok {
		glog.Errorf("expected an '*ingress.Location' type but %T was returned", input)
		return slash
	}

	path := location.Path
	if len(location.Rewrite.Target) > 0 && location.Rewrite.Target != path {
		if path == slash {
			return fmt.Sprintf("~* %s", path)
		}
		// baseuri regex will parse basename from the given location
		baseuri := `(?<baseuri>.*)`
		if !strings.HasSuffix(path, slash) {
			// Not treat the slash after "location path" as a part of baseuri
			baseuri = fmt.Sprintf(`\/?%s`, baseuri)
		}
		return fmt.Sprintf(`~* ^%s%s`, path, baseuri)
	}

	return path
}

// buildProxyPass produces the proxy pass string, if the ingress has redirects
// (specified through the nginx.ingress.kubernetes.io/rewrite-to annotation)
// If the annotation nginx.ingress.kubernetes.io/add-base-url:"true" is specified it will
// add a base tag in the head of the response from the service
func buildProxyPass(host string, b interface{}, loc interface{}) string {
	backends, ok := b.([]*ingress.Backend)
	if !ok {
		glog.Errorf("expected an '[]*ingress.Backend' type but %T was returned", b)
		return ""
	}

	location, ok := loc.(*ingress.Location)
	if !ok {
		glog.Errorf("expected a '*ingress.Location' type but %T was returned", loc)
		return ""
	}

	path := location.Path
	proto := "http"

	upstreamName := location.Backend
	for _, backend := range backends {
		if backend.Name == location.Backend {
			if backend.Secure {
				proto = "https"
			}

			break
		}
	}

	// defProxyPass returns the default proxy_pass, just the name of the upstream
	defProxyPass := fmt.Sprintf("proxy_pass %s://%s;", proto, upstreamName)
	// if the path in the ingress rule is equals to the target: no special rewrite
	if path == location.Rewrite.Target {
		return defProxyPass
	}

	if !strings.HasSuffix(path, slash) {
		path = fmt.Sprintf("%s/", path)
	}

	if len(location.Rewrite.Target) > 0 {
		abu := ""
		if location.Rewrite.AddBaseURL {
			// path has a slash suffix, so that it can be connected with baseuri directly
			bPath := fmt.Sprintf("%s%s", path, "$baseuri")
			regex := `(<(?:H|h)(?:E|e)(?:A|a)(?:D|d)(?:[^">]|"[^"]*")*>)`
			if len(location.Rewrite.BaseURLScheme) > 0 {
				abu = fmt.Sprintf(`subs_filter '%v' '$1<base href="%v://$http_host%v">' ro;
	    `, regex, location.Rewrite.BaseURLScheme, bPath)
			} else {
				abu = fmt.Sprintf(`subs_filter '%v' '$1<base href="$scheme://$http_host%v">' ro;
	    `, regex, bPath)
			}
		}

		xForwardedPrefix := ""
		if location.XForwardedPrefix {
			xForwardedPrefix = fmt.Sprintf(`proxy_set_header X-Forwarded-Prefix "%s";
	    `, path)
		}
		if location.Rewrite.Target == slash {
			// special case redirect to /
			// ie /something to /
			return fmt.Sprintf(`
	    rewrite %s(.*) /$1 break;
	    rewrite %s / break;
	    %vproxy_pass %s://%s;
	    %v`, path, location.Path, xForwardedPrefix, proto, upstreamName, abu)
		}

		return fmt.Sprintf(`
	    rewrite %s(.*) %s/$1 break;
	    %vproxy_pass %s://%s;
	    %v`, path, location.Rewrite.Target, xForwardedPrefix, proto, upstreamName, abu)
	}

	// default proxy_pass
	return defProxyPass
}

func isValidClientBodyBufferSize(input interface{}) bool {
	s, ok := input.(string)
	if !ok {
		glog.Errorf("expected an 'string' type but %T was returned", input)
		return false
	}

	if s == "" {
		return false
	}

	_, err := strconv.Atoi(s)
	if err != nil {
		sLowercase := strings.ToLower(s)

		kCheck := strings.TrimSuffix(sLowercase, "k")
		_, err := strconv.Atoi(kCheck)
		if err == nil {
			return true
		}

		mCheck := strings.TrimSuffix(sLowercase, "m")
		_, err = strconv.Atoi(mCheck)
		if err == nil {
			return true
		}

		glog.Errorf("client-body-buffer-size '%v' was provided in an incorrect format, hence it will not be set.", s)
		return false
	}

	return true
}

// TODO: Needs Unit Tests
func buildUpstreamName(host string, b interface{}, loc interface{}) string {
	location, ok := loc.(*ingress.Location)
	if !ok {
		glog.Errorf("expected a '*ingress.Location' type but %T was returned", loc)
		return ""
	}

	upstreamName := location.Backend
	return upstreamName
}

func buildForwardedFor(input interface{}) string {
	s, ok := input.(string)
	if !ok {
		glog.Errorf("expected a 'string' type but %T was returned", input)
		return ""
	}

	ffh := strings.Replace(s, "-", "_", -1)
	ffh = strings.ToLower(ffh)
	return fmt.Sprintf("$http_%v", ffh)
}

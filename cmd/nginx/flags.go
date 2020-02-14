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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/spf13/pflag"

	apiv1 "k8s.io/api/core/v1"

	"github.com/open-cluster-management/management-ingress/pkg/ingress/annotations/parser"
	"github.com/open-cluster-management/management-ingress/pkg/ingress/controller"
	ngx_config "github.com/open-cluster-management/management-ingress/pkg/ingress/controller/config"
	ing_net "github.com/open-cluster-management/management-ingress/pkg/net"
)

func parseFlags() (bool, *controller.Configuration, error) {
	var (
		flags = pflag.NewFlagSet("", pflag.ExitOnError)

		apiserverHost = flags.String("apiserver-host", "", "The address of the Kubernetes Apiserver "+
			"to connect to in the format of protocol://address:port, e.g., "+
			"http://localhost:8080. If not specified, the assumption is that the binary runs inside a "+
			"Kubernetes cluster and local discovery is attempted.")
		kubeConfigFile = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization and master location information.")

		configMap = flags.String("configmap", "",
			`Name of the ConfigMap that contains the custom configuration to use`)

		httpPort  = flags.Int("http-port", 8080, `Indicates the port to use for HTTP traffic`)
		httpsPort = flags.Int("https-port", 8443, `Indicates the port to use for HTTPS traffic`)

		showVersion = flags.Bool("version", false,
			`Shows release information about the NGINX Ingress controller`)

		resyncPeriod = flags.Duration("sync-period", 600*time.Second,
			`Relist and confirm cloud resources this often. Default is 10 minutes`)

		watchNamespace = flags.String("watch-namespace", apiv1.NamespaceAll,
			`Namespace to watch for Ingress. Default is to watch all namespaces`)

		annotationsPrefix = flags.String("annotations-prefix", "icp.management.ibm.com", `Prefix of the ingress annotations.`)

		syncRateLimit = flags.Float32("sync-rate-limit", 0.3,
			`Define the sync frequency upper limit`)

		defSSLCertificate = flags.String("default-ssl-certificate", "kube-system/router-certs", `Name of the secret
		that contains a SSL certificate to be used as default for a HTTPS catch-all server.
		Takes the form <namespace>/<secret name>.`)

		updateStatus = flags.Bool("update-status", true, `Indicates if the
		ingress controller should update the Ingress status IP/hostname. Default is true`)

		electionID = flags.String("election-id", "ingress-controller-leader", `Election id to use for status update.`)
	)

	flag.Set("logtostderr", "true")

	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args)
	flag.Set("logtostderr", "true")

	// Workaround for this issue:
	// https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})

	if *showVersion {
		return true, nil, nil
	}

	parser.AnnotationsPrefix = *annotationsPrefix

	// check port collisions
	if !ing_net.IsPortAvailable(*httpPort) {
		return false, nil, fmt.Errorf("Port %v is already in use. Please check the flag --http-port", *httpPort)
	}

	if !ing_net.IsPortAvailable(*httpsPort) {
		return false, nil, fmt.Errorf("Port %v is already in use. Please check the flag --https-port", *httpsPort)
	}

	config := &controller.Configuration{
		APIServerHost:         *apiserverHost,
		KubeConfigFile:        *kubeConfigFile,
		UpdateStatus:          *updateStatus,
		ElectionID:            *electionID,
		ResyncPeriod:          *resyncPeriod,
		Namespace:             *watchNamespace,
		ConfigMapName:         *configMap,
		SyncRateLimit:         *syncRateLimit,
		DefaultSSLCertificate: *defSSLCertificate,
		ListenPorts: &ngx_config.ListenPorts{
			HTTP:  *httpPort,
			HTTPS: *httpsPort,
		},
	}

	return false, config, nil
}

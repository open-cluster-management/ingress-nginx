/*
 Licensed Materials - Property of IBM
 (c) Copyright IBM Corporation 2018, 2019. All Rights Reserved.
 Note to U.S. Government Users Restricted Rights:
 Use, duplication or disclosure restricted by GSA ADP Schedule
 Contract with IBM Corp.

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

// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package class

import (
	"github.com/golang/glog"
	networking "k8s.io/api/networking/v1"
)

const (
	// IngressKey picks a specific "class" for the Ingress.
	// The controller only processes Ingresses with this annotation either
	// unset, or set to either the configured value or the empty string.
	IngressKey = "kubernetes.io/ingress.class"
)

var (
	// DefaultClass defines the default class used in the nginx ingres controller
	DefaultClass = "ingress-open-cluster-management"

	// IngressClass sets the runtime ingress class to use
	// An empty string means accept all ingresses without
	// annotation and the ones configured with class nginx
	IngressClass = "ingress-open-cluster-management"
)

// IsValid returns true if the given Ingress either doesn't specify
// the ingress.class annotation, or it's set to the configured in the
// ingress controller.
func IsValid(ing *networking.Ingress) bool {
	ingress, ok := ing.GetAnnotations()[IngressKey]
	if !ok {
		glog.V(3).Infof("annotation %v is not present in ingress %v/%v", IngressKey, ing.Namespace, ing.Name)
	}

	return ingress == IngressClass || ingress == DefaultClass
}

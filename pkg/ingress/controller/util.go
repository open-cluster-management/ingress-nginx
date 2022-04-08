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

// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package controller

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/golang/glog"
	api "k8s.io/api/core/v1"
	"k8s.io/klog"

	"github.com/stolostron/management-ingress/pkg/ingress"
)

// newUpstream creates an upstream without servers.
func newUpstream(name string) *ingress.Backend {
	return &ingress.Backend{
		Name:    name,
		Service: &api.Service{},
	}
}

// sysctlSomaxconn returns the value of net.core.somaxconn, i.e.
// maximum number of connections that can be queued for acceptance
// http://nginx.org/en/docs/http/ngx_http_core_module.html#listen
func sysctlSomaxconn() int {
	maxConns, err := getSysctl("net/core/somaxconn")
	if err != nil || maxConns < 512 {
		glog.V(3).Infof("system net.core.somaxconn=%v (using system default)", maxConns)
		return 511
	}

	return maxConns
}

// rlimitMaxNumFiles returns hard limit for RLIMIT_NOFILE
func rlimitMaxNumFiles() int {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		klog.Errorf("Error reading system maximum number of open file descriptors (RLIMIT_NOFILE): %v", err)
		return 0
	}
	klog.V(2).Infof("rlimit.max=%v", rLimit.Max)
	return int(rLimit.Max)
}

func intInSlice(i int, list []int) bool {
	for _, v := range list {
		if v == i {
			return true
		}
	}
	return false
}

// getSysctl returns the value for the specified sysctl setting
func getSysctl(sysctl string) (int, error) {
	data, err := ioutil.ReadFile(filepath.Clean(path.Join("/proc/sys", sysctl)))
	if err != nil {
		return -1, err
	}

	val, err := strconv.Atoi(strings.Trim(string(data), " \n"))
	if err != nil {
		return -1, err
	}

	return val, nil
}

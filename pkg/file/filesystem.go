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

// Copyright (c) 2021 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package file

import (
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
)

// Filesystem is an interface that we can use to mock various filesystem operations
type Filesystem interface {
	// from "os"
	Stat(name string) (os.FileInfo, error)
	Create(name string) (File, error)
	Rename(oldpath, newpath string) error
	MkdirAll(path string, perm os.FileMode) error
	Chtimes(name string, atime time.Time, mtime time.Time) error
	RemoveAll(path string) error
	Remove(name string) error

	// from "io/ioutil"
	ReadFile(filename string) ([]byte, error)
	TempDir(dir, prefix string) (string, error)
	TempFile(dir, prefix string) (File, error)
	ReadDir(dirname string) ([]os.FileInfo, error)
	Walk(root string, walkFn filepath.WalkFunc) error
}

// File is an interface that we can use to mock various filesystem operations typically
// accessed through the File object from the "os" package
type File interface {
	// for now, the only os.File methods used are those below, add more as necessary
	Name() string
	Write(b []byte) (n int, err error)
	Sync() error
	Close() error
}

// NewLocalFS implements Filesystem using same-named functions from "os" and "io/ioutil".
func NewLocalFS() (Filesystem, error) {
	fs := &DefaultFs{}

	err := initialize(false, fs)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// NewFakeFS creates an in-memory filesytem with all the required
// paths used by the ingress controller.
// This allows running test without polluting the local machine.
func NewFakeFS() (Filesystem, error) {
	fs := NewTempFs()

	err := initialize(true, fs)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// initialize creates the required directory structure and when
// runs as virtual filesystem it copies the local files to it
func initialize(isVirtual bool, fs Filesystem) error {
	for _, directory := range directories {
		err := fs.MkdirAll(directory, 0700)
		if err != nil {
			return err
		}
	}

	if !isVirtual {
		return nil
	}

	for _, file := range files {
		f, err := fs.Create(file)
		if err != nil {
			return err
		}

		_, err = f.Write([]byte(""))
		if err != nil {
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}
	}

	err := fs.MkdirAll("/proc", 0655)
	if err != nil {
		return err
	}

	glog.Info("Restoring generated (go-bindata) assets in virtual filesystem...")
	for _, assetName := range AssetNames() {
		err := restoreAsset("/", assetName, fs)
		if err != nil {
			return err
		}
	}

	return nil
}

// restoreAsset restores an asset under the given directory
func restoreAsset(dir, name string, fs Filesystem) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = fs.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}

	f, err := fs.Create(_filePath(dir, name))
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	//Missing info.Mode()

	err = fs.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

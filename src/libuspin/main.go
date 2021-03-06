//
// Copyright © 2016 Ikey Doherty <ikey@solus-project.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

// Package libuspin provides key utilities in dealing with the USpin
// configuration formats.
//
// Notably it provides the ImageSpec type, core to the main routines within
// USpin.
package libuspin

import (
	"errors"
	"fmt"
	"github.com/solus-project/libosdev/pkg"
	"libuspin/config"
	"libuspin/spec"
	"path/filepath"
	"strings"
)

var (
	// ErrNotEnoughOps should never, ever happen. So check for it. >_>
	ErrNotEnoughOps = errors.New("Internal error: 0 args passed to ApplyOperations")

	// ErrUnknownOperation is returned when we don't know how to handle an operation
	ErrUnknownOperation = errors.New("Unknown or unsupported operation requested")
)

// ImageSpec is a validated/loaded image configuration ready for building
type ImageSpec struct {
	Stack   *spec.OpStack
	Config  *config.ImageConfiguration
	BaseDir string // Used to join filename paths relative to the .spin file, i.e. packages
}

// NewImageSpec is a factory function to load a .spin file with it's associated
// Packages file prepped into a usable stack.
func NewImageSpec(spinFile string) (*ImageSpec, error) {
	is := &ImageSpec{}

	if !strings.HasSuffix(spinFile, ".spin") {
		return nil, fmt.Errorf("Not a .spin file: %v", spinFile)
	}

	// Try loading the configuration first
	conf, err := config.New(spinFile)
	if err != nil {
		return nil, err
	}

	// Grab the base directory from the .spin file
	is.BaseDir, err = filepath.Abs(filepath.Dir(spinFile))
	if err != nil {
		return nil, err
	}

	// Load packages file relative to the spin file
	parser := spec.NewParser()
	pkgsFile := filepath.Join(is.BaseDir, conf.Image.Packages)
	if err = parser.Parse(pkgsFile); err != nil {
		return nil, err
	}

	// Return new ImageSpec with our own copies
	return &ImageSpec{
		Stack:  parser.Stack,
		Config: conf,
	}, nil
}

// ApplyOperations will apply the given spec operations against the package
// manager instance
func ApplyOperations(manager pkg.Manager, ops []spec.Operation) error {
	if len(ops) == 0 {
		return ErrNotEnoughOps
	}
	switch ops[0].(type) {
	case *spec.OpRepo:
		// Insert one repo at a time
		for _, op := range ops {
			repo := op.(*spec.OpRepo)
			if err := manager.AddRepo(repo.RepoName, repo.RepoURI); err != nil {
				return err
			}
		}
		return nil
	case *spec.OpGroup:
		// Group/component goes in bulk
		ignoreSafety := ops[0].(*spec.OpGroup).IgnoreSafety
		var names []string
		for _, op := range ops {
			names = append(names, op.(*spec.OpGroup).GroupName)
		}
		return manager.InstallGroups(ignoreSafety, names)
	case *spec.OpPackage:
		// Group/component goes in bulk
		ignoreSafety := ops[0].(*spec.OpPackage).IgnoreSafety
		var names []string
		for _, op := range ops {
			names = append(names, op.(*spec.OpPackage).Name)
		}
		return manager.InstallPackages(ignoreSafety, names)
	default:
		return ErrUnknownOperation
	}
}

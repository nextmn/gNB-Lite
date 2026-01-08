// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT
package config

import (
	"io/ioutil"
	"net/netip"
	"path/filepath"

	"github.com/nextmn/json-api/jsonapi"

	"gopkg.in/yaml.v3"
)

func ParseConf(file string) (*GNBConfig, error) {
	var conf GNBConfig
	path, err := filepath.Abs(file)
	if err != nil {
		return nil, err
	}
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlFile, &conf)
	if err != nil {
		return nil, err
	}
	return &conf, nil
}

type GNBConfig struct {
	Control Control    `yaml:"control"`
	Ran     Ran        `yaml:"ran"`
	Cp      Cp         `yaml:"cp"`
	Logger  *Logger    `yaml:"logger,omitempty"`
	Gtp     netip.Addr `yaml:"gtp"`
}

type Control struct {
	Uri      jsonapi.ControlURI `yaml:"uri"`       // may contain domain name instead of ip address
	BindAddr netip.AddrPort     `yaml:"bind-addr"` // in the form `ip:port`
}

type Ran struct {
	BindAddr netip.AddrPort `yaml:"bind-addr"`
}

type Cp struct {
	Uri jsonapi.ControlURI `yaml:"uri"` // uri of the control plane
}

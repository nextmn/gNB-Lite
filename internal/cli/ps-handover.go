// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package cli

import (
	"net/http"
	"net/netip"

	"github.com/nextmn/json-api/jsonapi"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PsHandover struct {
	UeCtrl    jsonapi.ControlURI `json:"ue-ctrl"`
	GNBTarget jsonapi.ControlURI `json:"gnb-target"`
	Sessions  []netip.Addr       `json:"sessions"`
}

func (cli *Cli) PsHandover(c *gin.Context) {
	var ps PsHandover
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
	}
	go cli.HandlePsHandover(ps)
	c.Status(http.StatusNotImplemented)
}

func (cli *Cli) HandlePsHandover(ps PsHandover) {
}

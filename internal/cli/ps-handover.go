// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PsHandover struct {
	UeCtrl             jsonapi.ControlURI `json:"ue-ctrl"`
	GNBTarget          jsonapi.ControlURI `json:"gnb-target"`
	Sessions           []n1n2.Session     `json:"sessions"`
	IndirectForwarding bool               `json:"indirect-forwarding"`
}

func (cli *Cli) PsHandover(c *gin.Context) {
	var ps PsHandover
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
	}
	go cli.HandlePsHandover(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

func (cli *Cli) HandlePsHandover(ps PsHandover) {
	ctx := cli.PduSessions.Context()
	hr := n1n2.HandoverRequired{
		// Header
		SourcegNB: cli.PduSessions.Control,
		Cp:        cli.PduSessions.Cp,
		// Handover Required
		Ue:                 ps.UeCtrl,
		Sessions:           ps.Sessions,
		TargetgNB:          ps.GNBTarget,
		IndirectForwarding: ps.IndirectForwarding,
	}
	reqBody, err := json.Marshal(hr)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal n1n2.HandoverRequired")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cli.PduSessions.Cp.JoinPath("ps/handover-required").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create ps/handover-required")
		return
	}
	req.Header.Set("User-Agent", cli.PduSessions.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := cli.PduSessions.Client.Do(req); err != nil {
		logrus.WithError(err).Error("Could not send ps/handover-required")
		return
	}
}

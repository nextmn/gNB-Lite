// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"net/http"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/sirupsen/logrus"
)

type PsHandover struct {
	UeCtrl             jsonapi.ControlURI `json:"ue-ctrl"`
	GNBTarget          jsonapi.ControlURI `json:"gnb-target"`
	Sessions           []n1n2.Session     `json:"sessions"`
	IndirectForwarding bool               `json:"indirect-forwarding"`
}

func (cli Cli) PsHandover(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Cache-Control", "no-cache")
	var ps PsHandover
	if err := json.UnmarshalRead(req.Body, &ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		w.WriteHeader(http.StatusBadRequest)
		json.MarshalWrite(w, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	go cli.HandlePsHandover(ps)
	w.WriteHeader(http.StatusAccepted)
	json.MarshalWrite(w, jsonapi.Message{Message: "please refer to logs for more information"})
}

func (cli Cli) HandlePsHandover(ps PsHandover) {
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

	ctxDelay, cancel := context.WithTimeout(ctx, cli.PduSessions.CpDelay)
	defer cancel()
	select {
	case <-ctxDelay.Done():
		if err := ctx.Err(); err != nil {
			logrus.WithError(ctx.Err()).Error("Context was done before sending ps/handover-required")
			return
		}
		if _, err := cli.PduSessions.Client.Do(req); err != nil {
			logrus.WithError(err).Error("Could not send ps/handover-required")
		}
	}
}

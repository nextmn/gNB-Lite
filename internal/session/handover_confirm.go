// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package session

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (p *PduSessions) HandoverConfirm(c *gin.Context) {
	var ps n1n2.HandoverConfirm
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	logrus.WithFields(logrus.Fields{
		"ue": ps.UeCtrl.String(),
	}).Info("New Handover Confirm")
	go p.HandleHandoverConfirm(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

// Handover Confirm is send by the UE to the target gNB.
// Upon receiving Handover Confirm, the target gNB send a Handover Notify to the Control Plane.
func (p *PduSessions) HandleHandoverConfirm(ps n1n2.HandoverConfirm) {
	ctx := p.Context()
	// forward to CP
	resp := n1n2.HandoverNotify{
		// Header
		UeCtrl:    ps.UeCtrl,
		Cp:        ps.Cp,
		TargetGnb: ps.TargetGnb,
		// Handover Notify
		Sessions:  ps.Sessions,
		SourceGnb: ps.SourceGnb,
	}
	reqBody, err := json.Marshal(resp)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal n1n2.HandoverNotify")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Cp.JoinPath("ps/handover-notify").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create ps/handover-notify")
		return
	}
	req.Header.Set("User-Agent", p.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	ctxDelay, cancel := context.WithTimeout(ctx, p.CpDelay)
	defer cancel()
	select {
	case <-ctxDelay.Done():
		select {
		case <-ctx.Done():
			logrus.WithError(err).Error("Context was done before sending ps/handover-notify")
		default:
			if _, err := p.Client.Do(req); err != nil {
				logrus.WithError(err).Error("Could not send ps/handover-notify")
			}
		}
	}
}

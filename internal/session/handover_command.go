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

func (p *PduSessions) HandoverCommand(c *gin.Context) {
	var ps n1n2.HandoverCommand
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	logrus.WithFields(logrus.Fields{
		"ue": ps.UeCtrl.String(),
	}).Info("New Handover Command")
	go p.HandleHandoverCommand(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

// Handover Command is send to the source gNB by the Control Plane.
// Upon receiving an Handover Command, the source gNB configure temporary forwarding of DL traffic,
// and forward the Handover Command to the UE.
// PDU Session (including the forwarding of DL traffic) is removed with a timer.
func (p *PduSessions) HandleHandoverCommand(ps n1n2.HandoverCommand) {
	// Add forwarder for downlink
	for _, session := range ps.Sessions {
		if session.ForwardDownlinkFteid == nil || session.DownlinkFteid == nil {
			// TODO: notify CP of error
			continue
		}
		p.manager.ForwardDownlink[session.DownlinkFteid.Teid] = session.ForwardDownlinkFteid
		// TODO: remove downlink forward with a timer
		// TODO: remove pdu session after a timer
	}

	ctx := p.Context()
	// Forward to UE
	reqBody, err := json.Marshal(ps)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal n1n2.HandoverCommand")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ps.UeCtrl.JoinPath("ps/handover-command").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create ps/handover-command")
		return
	}
	req.Header.Set("User-Agent", p.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	ctxDelay, cancel := context.WithTimeout(ctx, p.UeDelay)
	defer cancel()
	select {
	case <-ctxDelay.Done():
		select {
		case <-ctx.Done():
			logrus.WithError(err).Error("Context was done before sending ps/handover-command")
		default:
			if _, err := p.Client.Do(req); err != nil {
				logrus.WithError(err).Error("Could not send ps/handover-command")
			}
		}
	}

}

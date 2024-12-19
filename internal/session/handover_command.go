// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package session

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func (s *PduSessions) HandoverCommand(c *gin.Context) {
	var ps n1n2.HandoverCommand
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	logrus.WithFields(logrus.Fields{
		"ue": ps.UeCtrl.String(),
	}).Info("New Handover Command")
	go s.HandleHandoverCommand(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

// Handover Command is send to the source gNB by the Control Plane.
// Upon receiving an Handover Command, the source gNB configure temporary forwarding of DL traffic,
// and forward the Handover Command to the UE.
// PDU Session (including the forwarding of DL traffic) is removed with a timer.
func (s *PduSessions) HandleHandoverCommand(ps n1n2.HandoverCommand) {
	// Add forwarder for downlink
	for _, session := range ps.Sessions {
		if session.ForwardDownlinkFteid == nil || session.DownlinkFteid == nil {
			// TODO: notify CP of error
			continue
		}
		s.manager.ForwardDownlink[session.DownlinkFteid.Teid] = session.ForwardDownlinkFteid
		// TODO: remove downlink forward with a timer
		// TODO: remove pdu session after a timer
	}

	ctx := s.Context()
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
	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := s.Client.Do(req); err != nil {
		logrus.WithError(err).Error("Could not send ps/handover-command")
		return
	}

}

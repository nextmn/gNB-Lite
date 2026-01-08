// Copyright Louis Royer and the NextMN contributors. All rights reserved.
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

func (s *PduSessions) HandoverRequest(c *gin.Context) {
	var ps n1n2.HandoverRequest
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	logrus.WithFields(logrus.Fields{
		"ue": ps.UeCtrl.String(),
	}).Info("New Handver Request")
	go s.HandleHandoverRequest(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

// Handover Request is send to the target gNB by the Control Plane.
// Upon receiving an Handover Request, the target gNB must allocate DL FTEID,
// and send it within an Handover Request Ack to the Control Plane.
// UL FTEID is included in Handover Request and the session
// can is pre-configured to be ready to be used as soon as Handover Notify is received
func (s *PduSessions) HandleHandoverRequest(ps n1n2.HandoverRequest) {
	ctx := s.Context()

	// allocate DL FTEIDs
	rsp_sessions := make([]n1n2.Session, len(ps.Sessions))
	copy(rsp_sessions, ps.Sessions)
	for i, session := range ps.Sessions {
		// allocate DL FTEID, and configure UL FTEID
		downlinkFTeid, err := s.manager.NewPduSession(ctx, session.Addr, ps.UeCtrl, session.UplinkFteid)
		if err != nil {
			logrus.WithError(err).Error("Could create PDU Session")
			// TODO: notify CP of the error
			return
		}
		rsp_sessions[i].DownlinkFteid = downlinkFTeid
	}

	// notify CP
	rsp := n1n2.HandoverRequestAck{
		// Header
		Cp:        ps.Cp,
		TargetgNB: ps.TargetgNB,
		// Handover Request Ack
		UeCtrl:    ps.UeCtrl,
		Sessions:  rsp_sessions,
		SourcegNB: ps.SourcegNB,
	}

	reqBody, err := json.Marshal(rsp)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal n1n2.HandoverRequestAck")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Cp.JoinPath("ps/handover-request-ack").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create ps/handover-request-ack")
		return
	}
	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := s.Client.Do(req); err != nil {
		logrus.WithError(err).Error("Could not send ps/handover-request-ack")
		return
	}
}

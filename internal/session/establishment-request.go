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

// request from UE
func (p *PduSessions) EstablishmentRequest(c *gin.Context) {
	// get PseReq
	var ps n1n2.PduSessionEstabReqMsg
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	logrus.WithFields(logrus.Fields{
		"ue": ps.Ue.String(),
	}).Info("New PDU Session establishment Request")
	go p.HandleEstablishmentRequest(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

func (p *PduSessions) HandleEstablishmentRequest(ps n1n2.PduSessionEstabReqMsg) {
	ctx := p.Context()
	// forward to cp
	reqBody, err := json.Marshal(ps)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal n1n2.PduSessionEstabReqMsg")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Cp.JoinPath("ps/establishment-request").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create ps/establishment-request")
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
			logrus.WithError(err).Error("Context was done before sending ps/establishment-request")
		default:
			if _, err := p.Client.Do(req); err != nil {
				logrus.WithError(err).Error("Could not send ps/establishment-request")
			}
		}
	}
}

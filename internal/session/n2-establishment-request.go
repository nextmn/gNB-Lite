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

// request from CP
func (p *PduSessions) N2EstablishmentRequest(c *gin.Context) {
	var ps n1n2.N2PduSessionReqMsg
	if err := c.BindJSON(&ps); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	logrus.WithFields(logrus.Fields{
		"ue":          ps.UeInfo.Header.Ue.String(),
		"upf":         ps.UplinkFteid.Addr,
		"uplink-teid": ps.UplinkFteid.Teid,
	}).Info("New PDU Session establishment Request")
	go p.HandleN2EstablishmentRequest(ps)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

func (p *PduSessions) HandleN2EstablishmentRequest(ps n1n2.N2PduSessionReqMsg) {
	ctx := p.Context()
	// allocate downlink teid
	downlinkFteid, err := p.manager.NewPduSession(ctx, ps.UeInfo.Addr, ps.UeInfo.Header.Ue, &ps.UplinkFteid)
	if err != nil {
		logrus.WithError(err).Error("Could create PDU Session")
		// TODO: notify CP of the error
		return
	}

	// send PseAccept to UE
	reqBody, err := json.Marshal(ps.UeInfo)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal UeInfo")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ps.UeInfo.Header.Ue.JoinPath("ps/establishment-accept").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create request for ps/establishment-accept")
		return
	}
	req.Header.Set("User-Agent", p.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := p.Client.Do(req); err != nil {
		logrus.WithError(err).Error("Could not send ps/establishment-accept")
		return
	}

	psresp := n1n2.N2PduSessionRespMsg{
		UeInfo:        ps.UeInfo,
		DownlinkFteid: *downlinkFteid,
	}
	// send N2PsResp to CP (with dl fteid)
	n2reqBody, err := json.Marshal(psresp)
	if err != nil {
		logrus.WithError(err).Error("Could not marshal n1n2.N2PduSessionRespMs")
		return
	}
	req2, err := http.NewRequestWithContext(ctx, http.MethodPost, ps.Cp.JoinPath("ps/n2-establishment-response").String(), bytes.NewBuffer(n2reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create request for ps/n2-establishment-response")
		return
	}
	req2.Header.Set("User-Agent", p.UserAgent)
	req2.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := p.Client.Do(req2); err != nil {
		logrus.WithError(err).Error("Could not create send request for ps/n2-establishment-response")
		return
	}

}

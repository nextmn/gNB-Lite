// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/netip"
	"sync"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PduSessions struct {
	PduSessionsMap sync.Map // key : UE 5G ip address; value: UE Control URI
	UserAgent      string
	Client         http.Client
	Control        jsonapi.ControlURI
	Cp             jsonapi.ControlURI
	GnbGtp         netip.Addr
	manager        *PduSessionsManager
}

func NewPduSessions(control jsonapi.ControlURI, cp jsonapi.ControlURI, manager *PduSessionsManager, userAgent string, gnbGtp netip.Addr) *PduSessions {
	return &PduSessions{
		Client:         http.Client{},
		PduSessionsMap: sync.Map{},
		UserAgent:      userAgent,
		Control:        control,
		Cp:             cp,
		GnbGtp:         gnbGtp,
		manager:        manager,
	}

}

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

	// forward to cp
	reqBody, err := json.Marshal(ps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not marshal json", Error: err})
		return
	}
	req, err := http.NewRequestWithContext(c, http.MethodPost, p.Cp.JoinPath("ps/establishment-request").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not create request", Error: err})
		return
	}
	req.Header.Set("User-Agent", p.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := p.Client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "no http response", Error: err})
		return
	}
	defer resp.Body.Close()
}

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
		"upf":         ps.Upf,
		"uplink-teid": ps.UplinkTeid,
	}).Info("New PDU Session establishment Request")
	// allocate downlink teid
	downlinkTeid, err := p.manager.NewPduSession(c, ps.UeInfo.Addr, ps.UeInfo.Header.Ue, ps.Upf, ps.UplinkTeid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could create PDU Session", Error: err})
		return
	}

	// send PseAccept to UE
	reqBody, err := json.Marshal(ps.UeInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not marshal json", Error: err})
		return
	}
	req, err := http.NewRequestWithContext(c, http.MethodPost, ps.UeInfo.Header.Ue.JoinPath("ps/establishment-accept").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not create request", Error: err})
		return
	}
	req.Header.Set("User-Agent", p.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := p.Client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "no http response", Error: err})
		return
	}
	defer resp.Body.Close()

	psresp := n1n2.N2PduSessionRespMsg{
		UeInfo:       ps.UeInfo,
		Gnb:          p.GnbGtp,
		DownlinkTeid: downlinkTeid,
	}
	// send N2PsResp to CP (with dl fteid)
	n2reqBody, err := json.Marshal(psresp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not marshal json", Error: err})
		return
	}
	req2, err := http.NewRequestWithContext(c, http.MethodPost, ps.Cp.JoinPath("ps/n2-establishment-response").String(), bytes.NewBuffer(n2reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not create request", Error: err})
		return
	}
	req2.Header.Set("User-Agent", p.UserAgent)
	req2.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp2, err := p.Client.Do(req2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "no http response", Error: err})
		return
	}
	defer resp2.Body.Close()

}

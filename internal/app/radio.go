// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sync"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Radio struct {
	peerMap   sync.Map // key:  UE Control URI, value: UE ran ip address
	Client    http.Client
	Control   jsonapi.ControlURI
	Data      netip.AddrPort
	UserAgent string
}

func NewRadio(control jsonapi.ControlURI, data netip.AddrPort, userAgent string) *Radio {
	return &Radio{
		peerMap:   sync.Map{},
		Client:    http.Client{},
		Control:   control,
		Data:      data,
		UserAgent: userAgent,
	}
}

func (r *Radio) Write(pkt []byte, srv *net.UDPConn, ue jsonapi.ControlURI) error {
	ueRan, ok := r.peerMap.Load(ue)
	if !ok {
		logrus.Trace("Unknown UE")
		return fmt.Errorf("Unknown UE")
	}

	_, err := srv.WriteToUDPAddrPort(pkt, ueRan.(netip.AddrPort))

	return err
}

// allow to peer to ue
func (r *Radio) Peer(c *gin.Context) {
	var peer n1n2.RadioPeerMsg
	if err := c.BindJSON(&peer); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	r.peerMap.Store(peer.Control, peer.Data)
	logrus.WithFields(logrus.Fields{
		"peer-control": peer.Control.String(),
		"peer-ran":     peer.Data,
	}).Info("New peer radio link")
	c.Status(http.StatusNoContent)
	msg := n1n2.RadioPeerMsg{
		Control: r.Control,
		Data:    r.Data,
	}

	reqBody, err := json.Marshal(msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not marshal json", Error: err})
		return
	}
	req, err := http.NewRequestWithContext(c, http.MethodPost, peer.Control.JoinPath("radio/peer").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "could not create request", Error: err})
		return
	}
	req.Header.Set("User-Agent", r.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := r.Client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, jsonapi.MessageWithError{Message: "no http response", Error: err})
		return
	}
	defer resp.Body.Close()

	// TODO: handle ue failure

	c.Status(http.StatusNoContent)
}

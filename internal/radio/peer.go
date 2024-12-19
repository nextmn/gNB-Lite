// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/nextmn/json-api/jsonapi"
	"github.com/nextmn/json-api/jsonapi/n1n2"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// allow to peer to ue
func (r *Radio) Peer(c *gin.Context) {
	var peer n1n2.RadioPeerMsg
	if err := c.BindJSON(&peer); err != nil {
		logrus.WithError(err).Error("could not deserialize")
		c.JSON(http.StatusBadRequest, jsonapi.MessageWithError{Message: "could not deserialize", Error: err})
		return
	}
	go r.HandlePeer(peer)
	c.JSON(http.StatusAccepted, jsonapi.Message{Message: "please refer to logs for more information"})
}

func (r *Radio) HandlePeer(peer n1n2.RadioPeerMsg) {
	ctx := r.Context()
	r.peerMap.Store(peer.Control.String(), peer.Data)
	logrus.WithFields(logrus.Fields{
		"peer-control": peer.Control.String(),
		"peer-ran":     peer.Data,
	}).Info("New peer radio link")
	msg := n1n2.RadioPeerMsg{
		Control: r.Control,
		Data:    r.Data,
	}

	reqBody, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Could not Marshal n1n2.RadioPeerMsg")
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, peer.Control.JoinPath("radio/peer").String(), bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.WithError(err).Error("Could not create radio/peer request")
		return
	}
	req.Header.Set("User-Agent", r.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if _, err := r.Client.Do(req); err != nil {
		logrus.WithError(err).Error("Could not send radio/peer request")
		return
	}
	// TODO: handle ue failure
}

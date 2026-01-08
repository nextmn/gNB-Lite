// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"net"
	"net/http"
	"net/netip"
	"sync"

	"github.com/nextmn/gnb-lite/internal/common"

	"github.com/nextmn/json-api/jsonapi"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Radio struct {
	common.WithContext

	peerMap   sync.Map // key:  UE Control URI (string), value: UE ran ip address
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
	ueRan, ok := r.peerMap.Load(ue.String())
	if !ok {
		logrus.Trace("Unknown UE")
		return ErrUnknownUE
	}

	_, err := srv.WriteToUDPAddrPort(pkt, ueRan.(netip.AddrPort))

	return err
}

func (r *Radio) Register(e *gin.Engine) {
	e.POST("/radio/peer", r.Peer)
}

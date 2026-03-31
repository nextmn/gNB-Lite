// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/nextmn/gnb-lite/internal/common"

	"github.com/nextmn/json-api/jsonapi"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Radio struct {
	common.WithContext

	peerMap   sync.Map // key:  UE Control URI (string), value: UE ran ip address
	delay     time.Duration
	Client    http.Client
	Control   jsonapi.ControlURI
	Data      netip.AddrPort
	UserAgent string
}

func NewRadio(control jsonapi.ControlURI, delay time.Duration, data netip.AddrPort, userAgent string) *Radio {
	return &Radio{
		peerMap:   sync.Map{},
		delay:     delay,
		Client:    http.Client{},
		Control:   control,
		Data:      data,
		UserAgent: userAgent,
	}
}

func (r *Radio) Write(ctx context.Context, pkt []byte, srv *net.UDPConn, ue jsonapi.ControlURI) error {
	radioCtx := r.Context()
	ueRan, ok := r.peerMap.Load(ue.String())
	if !ok {
		logrus.Trace("Unknown UE")
		return ErrUnknownUE
	}

	ctxDelay, cancel := context.WithTimeout(radioCtx, r.delay)
	defer cancel()
	select {
	case <-ctxDelay.Done():
		select {
		case <-radioCtx.Done():
			return radioCtx.Err()
		default:
			_, err := srv.WriteToUDPAddrPort(pkt, ueRan.(netip.AddrPort))
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}

}

func (r *Radio) Register(e *gin.Engine) {
	e.POST("/radio/peer", r.Peer)
}

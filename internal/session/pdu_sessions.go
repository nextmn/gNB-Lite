// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package session

import (
	"context"
	"net/http"
	"net/netip"
	"sync"

	"github.com/nextmn/json-api/jsonapi"

	"github.com/gin-gonic/gin"
)

type PduSessions struct {
	PduSessionsMap sync.Map // key : UE 5G ip address; value: UE Control URI
	UserAgent      string
	Client         http.Client
	Control        jsonapi.ControlURI
	Cp             jsonapi.ControlURI
	GnbGtp         netip.Addr
	manager        *PduSessionsManager

	// not exported because must not be modified
	ctx context.Context
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

func (p *PduSessions) Init(ctx context.Context) error {
	if ctx == nil {
		return ErrNilCtx
	}
	p.ctx = ctx
	return nil
}

func (p *PduSessions) Context() context.Context {
	if p.ctx != nil {
		return p.ctx
	}
	return context.Background()
}

func (p *PduSessions) Register(e *gin.Engine) {
	e.POST("/ps/establishment-request", p.EstablishmentRequest)
	e.POST("/ps/n2-establishment-request", p.N2EstablishmentRequest)
}

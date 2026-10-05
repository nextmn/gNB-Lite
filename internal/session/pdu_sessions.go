// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package session

import (
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/nextmn/gnb-lite/internal/common"

	"github.com/nextmn/json-api/jsonapi"
)

type PduSessions struct {
	common.WithContext

	PduSessionsMap sync.Map // key : UE 5G ip address; value: UE Control URI
	UserAgent      string
	Client         http.Client
	Control        jsonapi.ControlURI
	Cp             jsonapi.ControlURI
	CpDelay        time.Duration
	UeDelay        time.Duration
	GnbGtp         netip.Addr
	manager        *PduSessionsManager
}

func NewPduSessions(control jsonapi.ControlURI, cp jsonapi.ControlURI, cpDelay time.Duration, ueDelay time.Duration, manager *PduSessionsManager, userAgent string, gnbGtp netip.Addr) *PduSessions {
	return &PduSessions{
		Client:         http.Client{},
		PduSessionsMap: sync.Map{},
		UserAgent:      userAgent,
		Control:        control,
		Cp:             cp,
		CpDelay:        cpDelay,
		UeDelay:        ueDelay,
		GnbGtp:         gnbGtp,
		manager:        manager,
	}

}

func (p *PduSessions) Handler() http.Handler {
	sm := http.NewServeMux()
	sm.HandleFunc("POST /establishment-request", p.EstablishmentRequest)
	sm.HandleFunc("POST /n2-establishment-request", p.N2EstablishmentRequest)
	sm.HandleFunc("POST /handover-request", p.HandoverRequest)
	sm.HandleFunc("POST /handover-command", p.HandoverCommand)
	sm.HandleFunc("POST /handover-confirm", p.HandoverConfirm)
	return sm
}

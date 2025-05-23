// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"time"

	"github.com/nextmn/gnb-lite/internal/config"
	"github.com/nextmn/gnb-lite/internal/gtp"
	"github.com/nextmn/gnb-lite/internal/radio"
	"github.com/nextmn/gnb-lite/internal/session"
)

type Setup struct {
	config           *config.GNBConfig
	httpServerEntity *HttpServerEntity
	radio            *radio.Radio
	rDaemon          *radio.RadioDaemon
	psMan            *session.PduSessionsManager
	gtp              *gtp.Gtp
}

func NewSetup(config *config.GNBConfig) *Setup {
	r := radio.NewRadio(config.Control.Uri, config.Ran.BindAddr, "go-github-nextmn-gnb-lite")
	psMan := session.NewPduSessionsManager(config.Gtp)
	rDaemon := radio.NewRadioDaemon(r, psMan, config.Ran.BindAddr)
	ps := session.NewPduSessions(config.Control.Uri, config.Cp.Uri, psMan, "go-github-nextmn-gnb-lite", config.Gtp)
	return &Setup{
		config:           config,
		httpServerEntity: NewHttpServerEntity(config.Control.BindAddr, r, ps),
		radio:            r,
		rDaemon:          rDaemon,
		psMan:            psMan,
		gtp:              gtp.NewGtp(config.Gtp, psMan, rDaemon),
	}
}
func (s *Setup) Init(ctx context.Context) error {
	return nil
}

func (s *Setup) Run(ctx context.Context) error {
	if err := s.rDaemon.Start(ctx); err != nil {
		return err
	}
	if err := s.gtp.Start(ctx); err != nil {
		return err
	}
	if err := s.httpServerEntity.Start(ctx); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		ctxShutdown, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		s.httpServerEntity.WaitShutdown(ctxShutdown)
		s.gtp.WaitShutdown(ctxShutdown)
		s.rDaemon.WaitShutdown(ctxShutdown)
		return nil
	}
}

// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"fmt"
	"os/exec"
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
	routesInit       int // TODO: docker-setup
}

func NewSetup(config *config.GNBConfig) *Setup {
	r := radio.NewRadio(config.Control.Uri, config.Ran.OneWayDelays.Data, config.Ran.BindAddr, "go-github-nextmn-gnb-lite")
	psMan := session.NewPduSessionsManager(config.Gtp)
	rDaemon := radio.NewRadioDaemon(r, psMan, config.Ran.BindAddr)
	ps := session.NewPduSessions(config.Control.Uri, config.Cp.Uri, config.Cp.OneWayDelay, config.Ran.OneWayDelays.Control, psMan, "go-github-nextmn-gnb-lite", config.Gtp)
	return &Setup{
		config:           config,
		httpServerEntity: NewHttpServerEntity(config.Control.BindAddr, r, ps),
		radio:            r,
		rDaemon:          rDaemon,
		psMan:            psMan,
		gtp:              gtp.NewGtp(config.Gtp, psMan, rDaemon),
	}
}
func (s *Setup) waitShutdown(ctx context.Context) {
	// TODO: use waitGroup
	if s.httpServerEntity != nil {
		s.httpServerEntity.WaitShutdown(ctx)
	}
	if s.rDaemon != nil {
		s.rDaemon.WaitShutdown(ctx)
	}
	if s.gtp != nil {
		s.gtp.WaitShutdown(ctx)
	}
	if s.routesInit > 0 {
		s.cleanupRoutes(ctx)
	}
}

func (s *Setup) Run(ctx context.Context) error {
	defer func() {

		ctxShutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 1*time.Second)
		defer cancel()
		s.waitShutdown(ctxShutdown)
	}()

	if err := s.createRoutes(ctx); err != nil {
		return err
	}

	if err := s.rDaemon.Start(ctx); err != nil {
		return err
	}
	if err := s.gtp.Start(ctx); err != nil {
		return err
	}
	if err := s.httpServerEntity.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (s *Setup) createRoutes(ctx context.Context) error {
	// TODO: move this into github.com/nextmn/docker-setup
	for _, r := range s.config.DockerSetup.Routes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			cmd := exec.CommandContext(ctx, "ip", "route", "add", r.Prefix.String(), "via", r.Gateway.WithZone("").String(), "proto", "static")
			cmd.Env = []string{}
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("error running %s: %w", cmd.Args, err)
			}
			s.routesInit++
		}
	}
	return nil
}

func (s *Setup) cleanupRoutes(ctx context.Context) error {
	// TODO: move this into github.com/nextmn/docker-setup
	for i, r := range s.config.DockerSetup.Routes {
		if i >= s.routesInit { // cleanup only initialized routes
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			cmd := exec.CommandContext(ctx, "ip", "route", "del", r.Prefix.String(), "via", r.Gateway.WithZone("").String())
			cmd.Env = []string{}
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("error running %s: %w", cmd.Args, err)
			}
		}
	}
	return nil
}

// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/nextmn/gnb-lite/internal/cli"
	"github.com/nextmn/gnb-lite/internal/radio"
	"github.com/nextmn/gnb-lite/internal/session"

	"github.com/nextmn/json-api/healthcheck"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type HttpServerEntity struct {
	srv    *http.Server
	ps     *session.PduSessions
	radio  *radio.Radio
	closed chan struct{}
}

func NewHttpServerEntity(bindAddr netip.AddrPort, r *radio.Radio, ps *session.PduSessions) *HttpServerEntity {
	c := cli.NewCli(r, ps)
	// TODO: gin.SetMode(gin.DebugMode) / gin.SetMode(gin.ReleaseMode) depending on log level
	h := gin.Default()
	h.GET("/status", Status)

	// CLI
	c.Register(h)

	// Radio
	r.Register(h)

	// Pdu Sessions
	ps.Register(h)

	logrus.WithFields(logrus.Fields{"http-addr": bindAddr}).Info("HTTP Server created")
	e := HttpServerEntity{
		srv: &http.Server{
			Addr:    bindAddr.String(),
			Handler: h,
		},
		ps:     ps,
		radio:  r,
		closed: make(chan struct{}),
	}
	return &e
}

func (e *HttpServerEntity) Start(ctx context.Context) error {
	if err := e.ps.InitContext(ctx); err != nil {
		return err
	}

	l, err := net.Listen("tcp", e.srv.Addr)
	if err != nil {
		return err
	}
	go func(ln net.Listener) {
		logrus.Info("Starting HTTP Server")
		if err := e.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("Http Server error")
		}
	}(l)
	go func(ctx context.Context) {
		defer close(e.closed)
		select {
		case <-ctx.Done():
			ctxShutdown, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			if err := e.srv.Shutdown(ctxShutdown); err != nil {
				logrus.WithError(err).Info("HTTP Server Shutdown")
			}
		}
	}(ctx)
	return nil
}

func (e *HttpServerEntity) WaitShutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.closed:
		return nil
	}
}

// get status of the controller
func Status(c *gin.Context) {
	status := healthcheck.Status{
		Ready: true,
	}
	c.Header("Cache-Control", "no-cache")
	c.JSON(http.StatusOK, status)
}

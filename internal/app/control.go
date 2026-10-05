// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

import (
	"context"
	"encoding/json/v2"
	"net"
	"net/http"
	"net/netip"
	"time"

	"github.com/nextmn/gnb-lite/internal/cli"
	"github.com/nextmn/gnb-lite/internal/radio"
	"github.com/nextmn/gnb-lite/internal/session"

	"github.com/nextmn/json-api/healthcheck"
	"github.com/nextmn/logrus-formatter/httplog"

	"github.com/sirupsen/logrus"
)

type HttpServer struct {
	srv    *http.Server
	ps     *session.PduSessions
	closed chan struct{}
}

func NewHttpServer(bindAddr netip.AddrPort, r *radio.Radio, ps *session.PduSessions) *HttpServer {
	c := cli.Cli{Radio: r, PduSessions: ps}
	h := http.NewServeMux()
	h.HandleFunc("GET /status", Status)
	h.Handle("/cli", http.StripPrefix("/cli", c.Handler()))
	h.Handle("/radio", http.StripPrefix("/radio", r.Handler()))
	h.Handle("/ps", http.StripPrefix("/ps", ps.Handler()))
	logger := httplog.NewRequestLoggerMiddleware(h)

	logrus.WithFields(logrus.Fields{"http-addr": bindAddr}).Info("HTTP Server created")
	e := HttpServer{
		srv: &http.Server{
			Addr:    bindAddr.String(),
			Handler: logger,
		},
		ps:     ps,
		closed: make(chan struct{}),
	}
	return &e
}

func (s *HttpServer) Start(ctx context.Context) error {
	s.ps.InitContext(ctx)
	l, err := net.Listen("tcp", s.srv.Addr)
	if err != nil {
		return err
	}
	go func(ln net.Listener) {
		logrus.Info("Starting HTTP Server")
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("HTTP Server error")
		}
	}(l)
	go func(ctx context.Context) {
		defer close(s.closed)
		<-ctx.Done()
		ctxShutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 100*time.Millisecond)
		defer cancel()
		if err := s.srv.Shutdown(ctxShutdown); err == nil {
			logrus.Info("HTTP Server Shutdown")
		}
	}(ctx)
	return nil
}

func (s *HttpServer) WaitShutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return nil
	}
}

// get status of the controller
func Status(w http.ResponseWriter, req *http.Request) {
	status := healthcheck.Status{
		Ready: true,
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	json.MarshalWrite(w, status)
}

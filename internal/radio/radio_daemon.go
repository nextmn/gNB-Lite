// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/nextmn/gnb-lite/internal/session"

	"github.com/nextmn/json-api/jsonapi"

	"github.com/sirupsen/logrus"
)

const (
	TUN_MTU = 1400
)

type RadioDaemon struct {
	DlQueue            chan DLPkt
	radio              *Radio
	gnbRanAddr         netip.AddrPort
	PduSessionsManager *session.PduSessionsManager
	srv                *net.UDPConn
}

func NewRadioDaemon(radio *Radio, psMan *session.PduSessionsManager, gnbRanAddr netip.AddrPort) *RadioDaemon {
	return &RadioDaemon{
		DlQueue:            make(chan DLPkt),
		radio:              radio,
		PduSessionsManager: psMan,
		gnbRanAddr:         gnbRanAddr,
	}
}

func (r *RadioDaemon) runUplinkDaemon(ctx context.Context, srv *net.UDPConn) error {
	if srv == nil {
		logrus.Error("nil server")
		return fmt.Errorf("nil srv")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			buf := make([]byte, TUN_MTU)
			n, err := srv.Read(buf)
			if err != nil {
				logrus.WithError(err).Trace("error reading udp packet")
				return err
			}
			logrus.Trace("received new packet from ue")
			r.PduSessionsManager.WriteUplink(ctx, buf[:n])
		}
	}
	return nil
}

type DLPkt struct {
	Ue      jsonapi.ControlURI
	Payload []byte
}

func (r *RadioDaemon) WriteDownlink(payload []byte, ue jsonapi.ControlURI) error {
	if r.srv == nil {
		return fmt.Errorf("nil srv")
	}
	return r.radio.Write(payload, r.srv, ue)
}

func (r *RadioDaemon) Start(ctx context.Context) error {
	srv, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(r.gnbRanAddr))
	if err != nil {
		return err
	}
	r.srv = srv
	logrus.WithFields(logrus.Fields{
		"bind-addr": r.gnbRanAddr,
	}).Info("Starting Radio Simulatior")
	go func(ctx context.Context, srv *net.UDPConn) error {
		if srv == nil {
			return fmt.Errorf("nil srv")
		}
		select {
		case <-ctx.Done():
			srv.Close()
			return ctx.Err()
		}
		return nil
	}(ctx, srv)
	go func(ctx context.Context, srv *net.UDPConn) {
		defer srv.Close()
		r.runUplinkDaemon(ctx, srv)
	}(ctx, srv)
	return nil
}

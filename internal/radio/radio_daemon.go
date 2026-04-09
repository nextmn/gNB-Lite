// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"context"
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
	closed             chan struct{}
}

func NewRadioDaemon(radio *Radio, psMan *session.PduSessionsManager, gnbRanAddr netip.AddrPort) *RadioDaemon {
	return &RadioDaemon{
		DlQueue:            make(chan DLPkt),
		radio:              radio,
		PduSessionsManager: psMan,
		gnbRanAddr:         gnbRanAddr,
		closed:             make(chan struct{}),
	}
}

func (r *RadioDaemon) runUplinkDaemon(ctx context.Context, srv *net.UDPConn) error {
	if srv == nil {
		panic(errNilUdpConn)
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
			logrus.Trace("received new packet from UE")
			r.PduSessionsManager.WriteUplink(ctx, buf[:n])
		}
	}
}

type DLPkt struct {
	Ue      jsonapi.ControlURI
	Payload []byte
}

func (r *RadioDaemon) WriteDownlink(ctx context.Context, payload []byte, ue jsonapi.ControlURI) error {
	if r.srv == nil {
		panic(errNilUdpConn)
	}
	return r.radio.Write(ctx, payload, r.srv, ue)
}

func (r *RadioDaemon) Start(ctx context.Context) error {
	r.radio.InitContext(ctx)
	srv, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(r.gnbRanAddr))
	if err != nil {
		return err
	}
	r.srv = srv
	logrus.WithFields(logrus.Fields{
		"bind-addr": r.gnbRanAddr,
	}).Info("Starting Radio Simulator")
	go func(ctx context.Context, srv *net.UDPConn) error {
		if srv == nil {
			panic(errNilUdpConn)
		}
		<-ctx.Done()
		srv.Close()
		return ctx.Err()
	}(ctx, srv)
	go func(ctx context.Context, srv *net.UDPConn) {
		defer close(r.closed)
		defer srv.Close()
		r.runUplinkDaemon(ctx, srv)
	}(ctx, srv)
	return nil
}

func (r *RadioDaemon) WaitShutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.closed:
		return nil
	}
}

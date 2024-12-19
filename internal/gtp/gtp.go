// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package gtp

import (
	"context"
	"net"
	"net/netip"

	"github.com/nextmn/gnb-lite/internal/radio"
	"github.com/nextmn/gnb-lite/internal/session"

	"github.com/sirupsen/logrus"
	"github.com/wmnsk/go-gtp/gtpv1"
	"github.com/wmnsk/go-gtp/gtpv1/message"
)

type Gtp struct {
	ipAddr  netip.Addr
	psMan   *session.PduSessionsManager
	rDaemon *radio.RadioDaemon
	closed  chan struct{}
}

const GTPU_PORT = 2152

func NewGtp(ipAddr netip.Addr, psMan *session.PduSessionsManager, rDaemon *radio.RadioDaemon) *Gtp {
	return &Gtp{
		ipAddr:  ipAddr,
		psMan:   psMan,
		rDaemon: rDaemon,
		closed:  make(chan struct{}),
	}
}

func (gtp *Gtp) Start(ctx context.Context) error {
	logrus.WithFields(logrus.Fields{"listen-addr": gtp.ipAddr}).Info("Creating new GTP-U Protocol Entity")
	laddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(gtp.ipAddr, GTPU_PORT))
	uConn := gtpv1.NewUPlaneConn(laddr)
	uConn.DisableErrorIndication()
	uConn.AddHandler(message.MsgTypeTPDU, func(c gtpv1.Conn, senderAddr net.Addr, msg message.Message) error {
		return gtp.tpduHandler(ctx, c, senderAddr, msg)
	})
	go func(ctx context.Context) error {
		defer close(gtp.closed)
		defer uConn.Close()
		if err := uConn.ListenAndServe(ctx); err != nil {
			logrus.WithError(err).Trace("GTP uConn closed")
			return err
		}
		logrus.Trace("GTP uConn closed")
		return nil
	}(ctx)

	return nil
}

// handle GTP PDU (Downlink)
func (gtp *Gtp) tpduHandler(ctx context.Context, c gtpv1.Conn, senderAddr net.Addr, msg message.Message) error {
	teid := msg.TEID()
	// Try forwarding downlink (handover)
	if fd, err := gtp.psMan.GetForwarding(teid); err == nil {
		packet := msg.(*message.TPDU).Decapsulate()
		return gtp.psMan.ForwardUplink(ctx, packet, fd)
	}

	// Try to forward to UE over radio
	ue, err := gtp.psMan.GetUECtrl(teid)
	if err != nil {
		return err
	}
	packet := msg.(*message.TPDU).Decapsulate()
	return gtp.rDaemon.WriteDownlink(packet, ue)
}

func (gtp *Gtp) WaitShutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-gtp.closed:
		return nil
	}
}

// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package app

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

const GTPU_PORT = 2152

func (s *Setup) StartGtpUProtocolEntity(ctx context.Context, ipAddress netip.Addr) error {
	logrus.WithFields(logrus.Fields{"listen-addr": ipAddress}).Info("Creating new GTP-U Protocol Entity")
	laddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(ipAddress, GTPU_PORT))
	uConn := gtpv1.NewUPlaneConn(laddr)
	uConn.DisableErrorIndication()
	uConn.AddHandler(message.MsgTypeTPDU, func(c gtpv1.Conn, senderAddr net.Addr, msg message.Message) error {
		return tpduHandler(c, senderAddr, msg, s.psMan, s.rDaemon)
	})
	go func(ctx context.Context) error {
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
func tpduHandler(c gtpv1.Conn, senderAddr net.Addr, msg message.Message, psMan *session.PduSessionsManager, rDaemon *radio.RadioDaemon) error {
	teid := msg.TEID()
	ue, err := psMan.GetUECtrl(teid)
	if err != nil {
		return err
	}
	packet := msg.(*message.TPDU).Decapsulate()
	return rDaemon.WriteDownlink(packet, ue)
}

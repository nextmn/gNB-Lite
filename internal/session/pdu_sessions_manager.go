// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package session

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/nextmn/json-api/jsonapi"

	"github.com/sirupsen/logrus"
	"github.com/wmnsk/go-gtp/gtpv1"
	"github.com/wmnsk/go-gtp/gtpv1/message"
)

const GTPU_PORT = 2152

type PduSessionsManager struct {
	sync.Mutex

	Downlink map[uint32]jsonapi.ControlURI // teid: UE control uri
	Uplink   map[netip.Addr]*Fteid         // ue 5G ip address: uplink fteid
	GtpAddr  netip.Addr
	upfs     map[netip.Addr]*gtpv1.UPlaneConn
}

func NewPduSessionsManager(gtpAddr netip.Addr) *PduSessionsManager {
	return &PduSessionsManager{
		Downlink: make(map[uint32]jsonapi.ControlURI),
		Uplink:   make(map[netip.Addr]*Fteid),
		GtpAddr:  gtpAddr,
		upfs:     make(map[netip.Addr]*gtpv1.UPlaneConn),
	}
}

func (p *PduSessionsManager) WriteUplink(ctx context.Context, pkt []byte) error {
	if len(pkt) < 20 {
		logrus.Trace("too small to be an ipv4 packet")
		return fmt.Errorf("Too small to be an ipv4 packet")
	}
	if (pkt[0] >> 4) != 4 {
		logrus.Trace("not an ipv4 packet")
		return fmt.Errorf("Not an ipv4 packet")
	}
	src := netip.AddrFrom4([4]byte{pkt[12], pkt[13], pkt[14], pkt[15]})
	fteid, ok := p.Uplink[src]
	if !ok {
		logrus.WithFields(logrus.Fields{
			"ue": src,
		}).Trace("unknown UE")
		return fmt.Errorf("Unknown UE")
	}
	gpdu := message.NewHeaderWithExtensionHeaders(0x30, message.MsgTypeTPDU, fteid.Teid, 0, pkt, []*message.ExtensionHeader{}...)
	b, err := gpdu.Marshal()
	if err != nil {
		return err
	}
	uConn, ok := p.upfs[fteid.IpAddr]
	raddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(fteid.IpAddr, GTPU_PORT))
	if !ok {
		laddr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(p.GtpAddr, 0))
		uConn, err = gtpv1.DialUPlane(ctx, laddr, raddr)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"upf": raddr,
			}).Error("Failure to dial UPF")
			return err
		}
		p.upfs[fteid.IpAddr] = uConn
		go func(ctx context.Context, uConn *gtpv1.UPlaneConn) error {
			select {
			case <-ctx.Done():
				uConn.Close()
				return ctx.Err()
			}
			return nil
		}(ctx, uConn)
	}
	logrus.WithFields(logrus.Fields{
		"fteid": fteid,
	}).Trace("Forwarding packet to GTP")
	_, err = uConn.WriteTo(b, raddr)
	return err
}

func (p *PduSessionsManager) GetUECtrl(teid uint32) (jsonapi.ControlURI, error) {
	ueCtrl, ok := p.Downlink[teid]
	if !ok {
		return ueCtrl, fmt.Errorf("Unknown UE")
	}
	return ueCtrl, nil
}

type Fteid struct {
	IpAddr netip.Addr
	Teid   uint32
}

func (p *PduSessionsManager) NewPduSession(ctx context.Context, ueIpAddr netip.Addr, ueControlURI jsonapi.ControlURI, upf netip.Addr, uplinkTeid uint32) (uint32, error) {
	p.Lock()
	defer p.Unlock()

	ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(time.Millisecond*10)) // 10 ms should be more than enough…
	defer cancel()
	dlTeid, err := p.newTeidDl(ctxTimeout, ueControlURI)
	if err != nil {
		return dlTeid, err
	}
	p.Uplink[ueIpAddr] = &Fteid{
		IpAddr: upf,
		Teid:   uplinkTeid,
	}
	return dlTeid, err
}

// Warning: not thread safe
func (p *PduSessionsManager) newTeidDl(ctx context.Context, ueControlURI jsonapi.ControlURI) (uint32, error) {
	// teid are attributed randomly, and unique per pdu session
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			teid := rand.Uint32()
			if teid == 0 {
				continue // bad luck :(
			}
			if _, exists := p.Downlink[teid]; !exists {
				p.Downlink[teid] = ueControlURI
				return teid, nil
			}
		}
	}
}

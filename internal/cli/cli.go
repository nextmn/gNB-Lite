// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/nextmn/gnb-lite/internal/radio"
	"github.com/nextmn/gnb-lite/internal/session"

	"github.com/gin-gonic/gin"
)

type Cli struct {
	Radio       *radio.Radio
	PduSessions *session.PduSessions
}

func NewCli(r *radio.Radio, p *session.PduSessions) *Cli {
	return &Cli{
		Radio:       r,
		PduSessions: p,
	}
}

func (cli *Cli) Register(e *gin.Engine) {
	e.POST("/cli/ps/handover", cli.PsHandover)
}

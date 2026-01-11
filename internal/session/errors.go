// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package session

import (
	"errors"
)

var (
	ErrUnsupportedPDUType      = errors.New("unsupported PDU type")
	ErrPduSessionNotFound      = errors.New("PDU Session not found")
	ErrForwardDownlinkNotFound = errors.New("forward downlink rule not found")
)

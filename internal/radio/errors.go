// Copyright Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"errors"
)

var (
	// Programming error (used for panicking
	errNilUdpConn = errors.New("nil UDP Connection")

	ErrUnknownUE = errors.New("unknown UE")
)

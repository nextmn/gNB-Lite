// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package radio

import (
	"errors"
)

var (
	ErrNilUdpConn = errors.New("nil UDP Connection")
	ErrUnknownUE  = errors.New("Unknown UE")
)

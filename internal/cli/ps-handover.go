// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT

package cli

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (cli *Cli) PsHandover(c *gin.Context) {
	c.Status(http.StatusNotImplemented)
}

func (cli *Cli) HandlePsHandover() {
}

// Copyright 2024 Louis Royer and the NextMN contributors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.
// SPDX-License-Identifier: MIT
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/nextmn/json-api/healthcheck"
	"github.com/nextmn/logrus-formatter/logger"

	"github.com/nextmn/gnb-lite/internal/app"
	"github.com/nextmn/gnb-lite/internal/config"

	"github.com/adrg/xdg"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	logger.Init("NextMN-gNB Lite")
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	app := &cli.App{
		Name:                 "NextMN-gNB Lite",
		Usage:                "Experimental gNB Simulator",
		EnableBashCompletion: true,
		Authors: []*cli.Author{
			{Name: "Louis Royer"},
		},
		HideHelpCommand: true,
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				Usage:       "Load configuration from `FILE`",
				Required:    false,
				DefaultText: "${XDG_CONFIG_DIRS}/nextmn-gnb-lite/config.yaml",
				EnvVars:     []string{"CONFIG_FILE"},
			},
		},
		Before: func(ctx *cli.Context) error {
			if ctx.Path("config") == "" {
				if xdgPath, err := xdg.SearchConfigFile("nextmn-gnb-lite/config.yaml"); err != nil {
					cli.ShowAppHelp(ctx)
					logrus.WithError(err).Fatal("No configuration file defined")
				} else {
					ctx.Set("config", xdgPath)
				}
			}
			return nil
		},
		Action: func(ctx *cli.Context) error {
			conf, err := config.ParseConf(ctx.Path("config"))
			if err != nil {
				logrus.WithContext(ctx.Context).WithError(err).Fatal("Error loading config, exiting…")
			}
			if conf.Logger != nil {
				logrus.SetLevel(conf.Logger.Level)
			}

			if err := app.NewSetup(conf).Run(ctx.Context); err != nil {
				logrus.WithError(err).Fatal("Error while running, exiting…")
			}
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "healthcheck",
				Usage: "check status of the node",
				Action: func(ctx *cli.Context) error {
					conf, err := config.ParseConf(ctx.Path("config"))
					if err != nil {
						logrus.WithContext(ctx.Context).WithError(err).Fatal("Error loading config, exiting…")
					}
					if conf.Logger != nil {
						logrus.SetLevel(conf.Logger.Level)
					}
					if err := healthcheck.NewHealthcheck(*conf.Control.Uri.JoinPath("status"), "go-github-nextmn-gnb-lite").Run(ctx.Context); err != nil {
						os.Exit(1)
					}
					return nil
				},
			},
		},
	}
	if err := app.RunContext(ctx, os.Args); err != nil {
		logrus.Fatal(err)
	}
}

// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/private/process"
	"storj.io/private/version"
	"storj.io/storj/satellite"
	"storj.io/storj/satellite/satellitedb"
)

func cmdAdminRun(cmd *cobra.Command, args []string) (err error) {
	ctx, _ := process.Ctx(cmd)
	log := zap.L()

	runCfg.Debug.Address = *process.DebugAddrFlag

	identity, err := runCfg.Identity.Load()
	if err != nil {
		log.Error("Failed to load identity.", zap.Error(err))
		return errs.New("Failed to load identity: %+v", err)
	}

	db, err := satellitedb.New(log.Named("db"), runCfg.Database, satellitedb.Options{
		APIKeysLRUOptions: runCfg.APIKeysLRUOptions(),
	})
	if err != nil {
		return errs.New("Error starting master database on satellite api: %+v", err)
	}
	defer func() {
		err = errs.Combine(err, db.Close())
	}()

	peer, err := satellite.NewAdmin(log, identity, db, version.Build, &runCfg.Config, process.AtomicLevel(cmd))
	if err != nil {
		return err
	}

	_, err = peer.Version.Service.CheckVersion(ctx)
	if err != nil {
		return err
	}

	if err := process.InitMetricsWithCertPath(ctx, log, nil, runCfg.Identity.CertPath); err != nil {
		log.Warn("Failed to initialize telemetry batcher on satellite admin", zap.Error(err))
	}

	err = db.CheckVersion(ctx)
	if err != nil {
		log.Error("Failed satellite database version check.", zap.Error(err))
		return errs.New("Error checking version for satellitedb: %+v", err)
	}

	runError := peer.Run(ctx)
	closeError := peer.Close()
	return errs.Combine(runError, closeError)
}

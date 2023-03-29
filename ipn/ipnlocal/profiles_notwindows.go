// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

//go:build !windows

package ipnlocal

import (
	"context"
	"fmt"
	"runtime"

	"tailscale.com/ipn"
	"tailscale.com/version"
)

func (pm *profileManager) loadLegacyPrefs() (context.Context, ipn.PrefsView, error) {
	k := ipn.LegacyGlobalDaemonStateKey
	switch {
	case runtime.GOOS == "ios":
		k = "ipn-go-bridge"
	case version.IsSandboxedMacOS():
		k = "ipn-go-bridge"
	case runtime.GOOS == "android":
		k = "ipn-android"
	}
	prefs, err := pm.loadSavedPrefs(k)
	if err != nil {
		return nil, ipn.PrefsView{}, fmt.Errorf("calling ReadState on state store: %w", err)
	}
	pm.logf("migrating %q profile to new format", k)
	return context.Background(), prefs, nil
}

func (pm *profileManager) completeMigration(ctx context.Context) {
	// Do not delete the old state key, as we may be downgraded to an
	// older version that still relies on it.
}

// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package ipnlocal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"tailscale.com/atomicfile"
	"tailscale.com/ipn"
	"tailscale.com/util/winutil/policy"
)

const (
	legacyPrefsFile                  = "prefs"
	legacyPrefsMigrationSentinelFile = "_migrated-to-profiles"
	legacyPrefsExt                   = ".conf"
)

var errAlreadyMigrated = errors.New("profile migration already completed")

func legacyPrefsDir(uid ipn.WindowsUserID) (string, error) {
	// TODO(aaron): Ideally we'd have the impersonation token for the pipe's
	// client and use it to call SHGetKnownFolderPath, thus yielding the correct
	// path without having to make gross assumptions about directory names.
	usr, err := user.LookupId(string(uid))
	if err != nil {
		return "", err
	}
	if usr.HomeDir == "" {
		return "", fmt.Errorf("user %q does not have a home directory", uid)
	}
	userLegacyPrefsDir := filepath.Join(usr.HomeDir, "AppData", "Local", "Tailscale")
	return userLegacyPrefsDir, nil
}

type ctxKeyMigrationSentinelPath struct{}

func (pm *profileManager) loadLegacyPrefs() (context.Context, ipn.PrefsView, error) {
	userLegacyPrefsDir, err := legacyPrefsDir(pm.currentUserID)
	if err != nil {
		return nil, ipn.PrefsView{}, err
	}

	migrationSentinel := filepath.Join(userLegacyPrefsDir, legacyPrefsMigrationSentinelFile+legacyPrefsExt)
	// verify that migration sentinel is not present
	_, err = os.Stat(migrationSentinel)
	if err == nil {
		return nil, ipn.PrefsView{}, errAlreadyMigrated
	}
	if !os.IsNotExist(err) {
		return nil, ipn.PrefsView{}, err
	}

	prefsPath := filepath.Join(userLegacyPrefsDir, legacyPrefsFile+legacyPrefsExt)
	prefs, err := ipn.LoadPrefs(prefsPath)
	if err != nil {
		return nil, ipn.PrefsView{}, err
	}

	prefs.ControlURL = policy.SelectControlURL(defaultPrefs.ControlURL(), prefs.ControlURL)
	prefs.ExitNodeIP = resolveExitNodeIP(prefs.ExitNodeIP)
	prefs.ShieldsUp = resolveShieldsUp(prefs.ShieldsUp)
	prefs.ForceDaemon = resolveForceDaemon(prefs.ForceDaemon)

	pm.logf("migrating Windows profile to new format")
	ctx := context.WithValue(context.Background(), ctxKeyMigrationSentinelPath{}, migrationSentinel)
	return ctx, prefs.View(), nil
}

func (pm *profileManager) completeMigration(ctx context.Context) {
	migrationSentinel := ctx.Value(ctxKeyMigrationSentinelPath{}).(string)
	atomicfile.WriteFile(migrationSentinel, []byte{}, 0600)
}

func resolveShieldsUp(defval bool) bool {
	pol := policy.GetPreferenceOptionPolicy("AllowIncomingConnections")
	return !pol.ShouldEnable(!defval)
}

func resolveForceDaemon(defval bool) bool {
	pol := policy.GetPreferenceOptionPolicy("UnattendedMode")
	return pol.ShouldEnable(defval)
}

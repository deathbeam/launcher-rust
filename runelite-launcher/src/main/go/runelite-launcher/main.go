/*
 * Copyright (c) 2018, Tomas Slusny <slusnucky@gmail.com>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 * 1. Redistributions of source code must retain the above copyright notice, this
 *    list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR
 * ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 * SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */
package main

import (
	"fmt"
	"github.com/andlabs/ui"
	"github.com/mitchellh/go-homedir"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

func boot() {
	// Build system name
	systemName := runtime.GOOS

	if systemName != "darwin" {
		switch runtime.GOARCH {
		case "386":
			systemName += "32"
		case "amd64":
			systemName += "64"
		}
	}

	// Parse bootstrap properties
	bootstrap := ReadBootstrap("http://static.runelite.net/bootstrap.json")
	clientArtifactName := bootstrap.Client.ArtifactId
	clientArtifactVersion := bootstrap.Client.Version
	clientJarName := fmt.Sprintf("%s-%s-shaded.jar", clientArtifactName, clientArtifactVersion)

	// TODO: Parse distribution properties from somewhere
	distributionArtifactName := "runelite-distribution"
	distributionArtifactVersion := "1.0.0-SNAPSHOT"
	distributionDirName := fmt.Sprintf("%s-%s",
		distributionArtifactName,
		distributionArtifactVersion)
	distributionJarName := fmt.Sprintf("%s-%s.jar",
		distributionArtifactName,
		distributionArtifactVersion)
	distributionArchiveName := fmt.Sprintf("%s-%s-archive-distribution-%s.tar.gz",
		distributionArtifactName,
		distributionArtifactVersion,
		systemName)

	// Setup cache directories
	home, _ := homedir.Dir()
	runeliteHome := path.Join(home, ".runelite")
	launcherCache := path.Join(runeliteHome, "cache")
	systemCache := path.Join(launcherCache, systemName)
	distributionCache := path.Join(systemCache, distributionDirName)
	AppLog("System cache directory: %s", systemCache)

	// Setup versions
	distributionCacheVersionPath := path.Join(launcherCache, ".version-distribution")
	distributionCacheVersion := ReadVersion(distributionCacheVersionPath)
	clientCacheVersionPath := path.Join(launcherCache, ".version-client")
	clientCacheVersion := ReadVersion(clientCacheVersionPath)

	// TODO: Try to download distribution if not already downloaded
	distributionArchiveDestination := path.Join(launcherCache, distributionArchiveName)

	// Try to extract distribution if not already extracted
	if !FileExists(systemCache) || !CompareVersion(distributionCacheVersion, distributionArtifactVersion) {
		os.RemoveAll(systemCache)
		os.MkdirAll(systemCache, os.ModePerm)
		ExtractFile(distributionArchiveDestination, systemCache)
		SaveVersion(distributionCacheVersionPath, distributionArtifactVersion)
	}

	// Try to download shaded jar if not already present
	distributionPath := distributionCache

	if systemName == "darwin" {
		distributionPath = path.Join(distributionPath, "Contents", "Resources")
	}

	distributionJarDestination := path.Join(distributionPath, distributionJarName)

	if !FileExists(distributionJarDestination) || !CompareVersion(clientCacheVersion, clientArtifactVersion) {
		baseUrl := "http://repo.runelite.net/"
		groupPath := strings.Replace(bootstrap.Client.GroupId, ".", "/", -1)
		shadedJarUrl := fmt.Sprintf("%s/%s/%s/%s/%s",
			baseUrl, groupPath, clientArtifactName, clientArtifactVersion, clientJarName)

		os.RemoveAll(distributionJarDestination)

		DownloadFile(shadedJarUrl, distributionJarDestination, func(percent float64) {
			UpdateProgress(percent)
		})

		SaveVersion(clientCacheVersionPath, clientArtifactVersion)
	}

	// Build path to application executable
	distributionNativePath := distributionCache

	if systemName == "darwin" {
		distributionNativePath = path.Join(distributionNativePath, "Contents", "MacOS", distributionArtifactName)
	} else if strings.Contains(systemName, "windows") {
		distributionNativePath = path.Join(distributionNativePath, distributionArtifactName+".exe")
	} else {
		distributionNativePath = path.Join(distributionNativePath, distributionArtifactName)
	}

	// Launch application
	AppLog("Launching %s\n", distributionNativePath)
	cmd := exec.Command(distributionNativePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	HideWindow()
	err := cmd.Run()
	CloseWindow()

	if err != nil {
		panic(err)
	}
}

func main() {
	err := ui.Main(BuildGui(boot))

	if err != nil {
		panic(err)
	}
}

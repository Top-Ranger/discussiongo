// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2021,2022 Marcus Soll
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main implements the main server
package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
	"github.com/Top-Ranger/discussiongo/files"
)

func printInfo() {
	log.Println("DiscussionGo!")
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		log.Print("- no build info found")
		return
	}

	log.Printf("- go version: %s", bi.GoVersion)
	for _, s := range bi.Settings {
		switch s.Key {
		case "-tags":
			log.Printf("- build tags: %s", s.Value)
		case "vcs.revision":
			l := 7
			if len(s.Value) > 7 {
				s.Value = s.Value[:l]
			}
			log.Printf("- commit: %s", s.Value)
		case "vcs.modified":
			log.Printf("- files modified: %s", s.Value)
		}
	}
}

func main() {
	printInfo()

	// Set Translation
	SetDefaultTranslation(config.Language)

	// Init databases
	err := database.InitDB(config.DatabaseConfig)
	if err != nil {
		panic(err)
	}

	err = accesstimes.InitDB(config.DatabaseConfig)
	if err != nil {
		panic(err)
	}

	err = files.InitDB(config.DatabaseConfig)
	if err != nil {
		panic(err)
	}

	err = events.InitDB(config.DatabaseConfig)
	if err != nil {
		panic(err)
	}

	// Test SYSTEM
	exists, err := database.UserExists("SYSTEM")

	if err != nil {
		panic(err)
	}

	if !exists {
		pw := make([]byte, 33)
		_, err = rand.Read(pw)
		if err != nil {
			panic(err)
		}
		password := base64.StdEncoding.EncodeToString(pw)
		err = database.AddUser("SYSTEM", password, true)
		if err != nil {
			panic(err)
		}
		log.Printf("\nCreated SYSTEM\n\tUsername: SYSTEM\n\tPasswort: %s\n", password)
	} else {
		pw := make([]byte, 33)
		_, err = rand.Read(pw)
		if err != nil {
			panic(err)
		}
		password := base64.StdEncoding.EncodeToString(pw)
		err = database.EditPassword("SYSTEM", password)
		if err != nil {
			panic(err)
		}
		log.Printf("\nUpdated SYSTEM\n\tUsername: SYSTEM\n\tPasswort: %s\n", password)
	}
	err = database.SetAdmin("SYSTEM", true)
	if err != nil {
		panic(err)
	}

	err = startAdminDeleteLoop(config.AdminEventDuration)
	if err != nil {
		panic(err)
	}

	log.Println("Starting server at", config.Address)
	log.Fatal(http.ListenAndServe(config.Address, nil))
}

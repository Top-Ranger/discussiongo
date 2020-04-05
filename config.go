// SPDX-License-Identifier: Apache-2.0
// Copyright 2020 Marcus Soll
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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"
)

type configData struct {
	CanRegister             bool
	CanReadWithoutRegister  bool
	Address                 string
	InvitationAdmin         bool
	InvitationUser          bool
	ServerPrefix            string
	ServerPath              string
	CookieLogin             string
	CookieAuth              string
	CookieMinutes           int
	LengthPassword          int
	CreateInvitationMessage string
	ForumName               string
}

var config = configData{}
var authentificationDuration = 0 * time.Minute

func init() {
	c, err := loadConfig("./config.json")
	if err != nil {
		panic(err)
	}
	config = c
	authentificationDuration = time.Duration(c.CookieMinutes) * time.Minute
}

func loadConfig(path string) (configData, error) {
	log.Println("Loading config")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return configData{}, errors.New(fmt.Sprintln("Can not read config.json:", err))
	}

	c := configData{}
	err = json.Unmarshal(b, &c)
	if err != nil {
		return configData{}, errors.New(fmt.Sprintln("Error while parsing config.json:", err))
	}

	// sanity checks
	c.ServerPath = strings.TrimSuffix(c.ServerPath, "/")
	c.ServerPrefix = strings.TrimSuffix(c.ServerPrefix, "/")

	return c, nil
}

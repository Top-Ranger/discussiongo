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
	"net/http"

	"github.com/Top-Ranger/discussiongo/database"
)

type updateTopicPost struct {
	LastUpdate int64
}

func init() {
	http.HandleFunc("/updateTopicPost.json", updateTopicPostHandleFunc)
}

func updateTopicPostHandleFunc(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	u := updateTopicPost{database.GetLastUpdateTopicPost()}
	b, err := json.Marshal(&u)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	rw.Write(b)
}

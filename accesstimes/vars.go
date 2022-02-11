// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Marcus Soll
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

// Package accesstimes is responsible for saving the access times of users of topics.
package accesstimes

import (
	"database/sql"
	"time"
)

var (
	db         *sql.DB
	saveTime   = make(chan save, 10)
	deleteUser = make(chan string)
)

type save struct {
	Name  string
	Topic int
	Time  time.Time
}

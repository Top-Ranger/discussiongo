//go:build mysql

// SPDX-License-Identifier: Apache-2.0
// Copyright 2021,2022 Marcus Soll
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

package accesstimes

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const databaseVersion = "MySQL-1"

// InitDB initialises the database.
// Must be called before any other function.
// Config expects a DSN.
func InitDB(config string) error {
	newDb, err := sql.Open("mysql", config)
	if err != nil {
		return fmt.Errorf("accesstimes: can not open '%s': %w", config, err)
	}
	db = newDb
	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	go worker()
	return nil
}

// WaitWrite waits for an asynchronous write to finish.
// It must be used if it is required that new values from SaveTime are returned in GetTimes / GetUserTimes.
func WaitWrite() {
}

func worker() {
	for {
		select {
		case x := <-saveTime:
			_, err := db.Exec("REPLACE INTO times VALUES (?, ?, ?)", x.Name, x.Topic, x.Time.Unix())
			if err != nil {
				log.Println("Can not insert access time:", err)
			}
		case _ = <-deleteUser:
		}
	}
}

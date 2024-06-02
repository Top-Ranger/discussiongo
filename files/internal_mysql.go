//go:build mysql

// SPDX-License-Identifier: Apache-2.0
// Copyright 2021,2022,2024 Marcus Soll
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

package files

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const databaseVersion = "MySQL-2"

// InitDB initialises the database.
// Must be called before any other function.
// Config expects a DSN.
func InitDB(config string) error {
	newDb, err := sql.Open("mysql", config)
	if err != nil {
		return fmt.Errorf("files: can not open '%s': %w", config, err)
	}

	// Check version
	rows, err := newDb.Query("SELECT value FROM meta WHERE mkey=?", "version")
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return fmt.Errorf("database has no version")
	}

	var version string
	err = rows.Scan(&version)
	if err != nil {
		return err
	}

	if version != databaseVersion {
		return fmt.Errorf("database is %s, should be %s", version, databaseVersion)
	}

	// Everything ok
	db = newDb
	db.SetConnMaxLifetime(time.Minute * 1)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	return nil
}

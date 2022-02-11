//go:build sqlite

// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2022 Marcus Soll
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
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // Database driver
)

var waitWrite = sync.Cond{}

// InitDB initialises the database.
// Must be called before any other function.
// SQLite will ignore all config.
func InitDB(config string) error {
	err := connectToDB("./accesstimes.sqlite3")
	go worker()
	return err
}

// WaitWrite waits for an asynchronous write to finish.
// It must be used if it is required that new values from SaveTime are returned in GetTimes / GetUserTimes.
func WaitWrite() {
	waitWrite.L.Lock()
	defer waitWrite.L.Unlock()
	waitWrite.Wait()
}

func init() {
	waitWrite.L = new(sync.Mutex)
}

// connectToDB returns a sql.DB object connected to the sqlite file given by path.
// If the file doesn't exist, it will be created (including database schema).
func connectToDB(path string) error {
	// Check if file exists
	newFile := false
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		newFile = true
	} else if err != nil {
		return err
	}

	// Open database
	newDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	// Create tables if needed
	if newFile {
		tx, err := newDB.Begin()
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE TABLE meta (key TEXT NOT NULL PRIMARY KEY, value TEXT)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("INSERT INTO meta VALUES ('version', 1)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("PRAGMA secure_delete=ON")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE TABLE times (name TEXT NOT NULL, topic INT, time INT, PRIMARY KEY(name, topic))")
		if err != nil {
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	} else {
		// Get version number
		var versionNr int

		rows, err := newDB.Query("SELECT value FROM meta WHERE key='version'")
		if err != nil {
			return err
		}

		defer rows.Close()
		if !rows.Next() {
			return err
		}

		err = rows.Scan(&versionNr)
		if err != nil {
			return err
		}

		// We need to close now - or else the database will be locked later when we try to modify the database the next step
		rows.Close()

		log.Println("Detected access time database version", versionNr)

		// Upgrade
		switch versionNr {
		default:
			log.Println("Database is on newest version")
		}
	}

	db = newDB
	return nil
}

func worker() {
	buffer := make([]*save, 0, 100)
	t := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-t.C:
			if len(buffer) != 0 {
				tx, err := db.Begin()
				if err != nil {
					log.Println("Can not begin transaction:", err)
				}
				for i := range buffer {
					if buffer[i] == nil {
						continue
					}
					_, err := tx.Exec("INSERT OR REPLACE INTO times VALUES (?, ?, ?)", buffer[i].Name, buffer[i].Topic, buffer[i].Time.Unix())
					if err != nil {
						log.Println("Can not insert access time:", err)
					}
				}
				err = tx.Commit()
				if err != nil {
					log.Println("Can not commit transaction:", err)
				}
				waitWrite.Broadcast()
				buffer = make([]*save, 0, len(buffer)*2+10)
			}

		case x := <-saveTime:
			buffer = append(buffer, &x)

		case u := <-deleteUser:
			for i := range buffer {
				if buffer[i] == nil {
					continue
				}
				if buffer[i].Name == u {
					buffer[i] = nil
				}
			}
		}
	}
}

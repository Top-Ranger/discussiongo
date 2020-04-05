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

// Package database provides the internal database of DiscussionGo!
// The access times of topics are in an other module (accesstimes). This allows for both data sources to be operated separately.
package database

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // Database driver
)

var (
	db *sql.DB
)

func init() {
	err := connectToDB("./database.sqlite3")
	if err != nil {
		panic(err)
	}
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

		_, err = tx.Exec("INSERT INTO meta VALUES ('version', 5)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("PRAGMA foreign_keys=ON")
		if err != nil {
			return err
		}

		_, err = tx.Exec("PRAGMA secure_delete=ON")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE TABLE user (name TEXT NOT NULL PRIMARY KEY, salt TEXT, encodedpasswort TEXT, admin BOOLEAN, comment TEXT DEFAULT '', invitedby TEXT DEFAULT '', invitationdirect BOOL DEFAULT 0, lastseen INTEGER DEFAULT 0)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE TABLE topic (id INTEGER PRIMARY KEY, name TEXT, creator TEXT, created INTEGER, lastmodified INTEGER, closed BOOL DEFAULT 0, pinned BOOL DEFAULT 0)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE INDEX idx_topic_lastmodified_desc ON topic (lastmodified DESC)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE TABLE post (id INTEGER PRIMARY KEY, content TEXT, poster TEXT, time INTEGER, topic INTEGER, FOREIGN KEY(topic) REFERENCES topic(id) ON UPDATE CASCADE ON DELETE CASCADE)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE INDEX idx_post_topic_time_asc ON post (topic, time ASC)")
		if err != nil {
			return err
		}

		_, err = tx.Exec("CREATE TABLE invitations (id TEXT NOT NULL PRIMARY KEY, creator TEXT)")
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

		log.Println("Detected database version", versionNr)

		// Upgrade
		switch versionNr {
		case 1:
			log.Println("Upgrade database 1 -> 2")

			tx, err := newDB.Begin()
			if err != nil {
				return err
			}

			_, err = tx.Exec("ALTER TABLE user ADD COLUMN comment TEXT DEFAULT ''")
			if err != nil {
				return err
			}

			_, err = tx.Exec("ALTER TABLE user ADD COLUMN invitedby TEXT DEFAULT ''")
			if err != nil {
				return err
			}

			_, err = tx.Exec("ALTER TABLE user ADD COLUMN invitationdirect BOOL DEFAULT 0")
			if err != nil {
				return err
			}

			_, err = tx.Exec("UPDATE meta SET value=2 WHERE key='version'")
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}

			log.Println("Upgrade done")
			fallthrough

		case 2:
			log.Println("Upgrade database 2 -> 3")

			tx, err := newDB.Begin()
			if err != nil {
				return err
			}

			_, err = tx.Exec("ALTER TABLE topic ADD COLUMN closed BOOL DEFAULT 0")
			if err != nil {
				return err
			}

			_, err = tx.Exec("ALTER TABLE topic ADD COLUMN pinned BOOL DEFAULT 0")
			if err != nil {
				return err
			}

			_, err = tx.Exec("UPDATE meta SET value=3 WHERE key='version'")
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}

			log.Println("Upgrade done")
			fallthrough
		case 3:
			log.Println("Upgrade database 3 -> 4")

			tx, err := newDB.Begin()
			if err != nil {
				return err
			}

			_, err = tx.Exec("ALTER TABLE user ADD COLUMN lastseen INTEGER DEFAULT 0")
			if err != nil {
				return err
			}

			_, err = tx.Exec("UPDATE meta SET value=4 WHERE key='version'")
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}

			log.Println("Upgrade done")
			fallthrough
		case 4:
			log.Println("Upgrade database 4 -> 5")

			tx, err := newDB.Begin()
			if err != nil {
				return err
			}

			_, err = tx.Exec("CREATE INDEX idx_post_topic_time_asc ON post (topic, time ASC)")
			if err != nil {
				return err
			}

			_, err = tx.Exec("CREATE INDEX idx_topic_lastmodified_desc ON topic (lastmodified DESC)")
			if err != nil {
				return err
			}

			_, err = tx.Exec("UPDATE meta SET value=5 WHERE key='version'")
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}

			log.Println("Upgrade done")
			fallthrough
		default:
			log.Println("Database is on newest version")
		}
	}

	db = newDB
	return nil
}

// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Marcus Soll
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

package authtoken

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var cleanupStarted = sync.Once{}

// Authtoken represents an authtoken for user validation.
type Authtoken struct {
	ID         string
	User       string
	ValidUntil time.Time
}

// DeleteToken removes a single token fom database.
func DeleteToken(authtoken string) error {
	_, err := db.Exec("DELETE FROM authtoken WHERE id=?", authtoken)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}
	return nil
}

// DeleteUserToken removes all authtoken associated by a user.
// It returns the number of deleted tokens.
func DeleteUserToken(user string) (int64, error) {
	r, err := db.Exec("DELETE FROM authtoken WHERE user=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}
	return count, nil
}

// GetNewToken inserts an authtoken into the database and returns it.
// The token will be generated uniquely.
func GetNewToken(user string, minutesValid int) (Authtoken, error) {
	b := make([]byte, 35)
	_, err := rand.Read(b)
	if err != nil {
		return Authtoken{}, err
	}
	validUntil := time.Now().Add(time.Duration(minutesValid) * time.Minute)
	authtoken := base32.StdEncoding.EncodeToString(b)
	intDate := validUntil.Unix()

	_, err = db.Exec("INSERT INTO authtoken (id, user, validUntil) VALUES (?, ?, ?)", authtoken, user, intDate)
	if err != nil {
		return Authtoken{}, err
	}

	return Authtoken{
		ID:         authtoken,
		User:       user,
		ValidUntil: validUntil,
	}, nil

}

// CheckUser checks whether an authstring belongs to a user and the user is valid.
// It will also check valid until.
func CheckUser(authtoken string) (string, time.Time, bool) {
	rows, err := db.Query("SELECT user,validUntil FROM authtoken WHERE id=?", authtoken)
	if err != nil {
		return "", time.Time{}, false
	}
	defer rows.Close()

	if !rows.Next() {
		return "", time.Time{}, false
	}

	var user string
	var intDate int64
	err = rows.Scan(&user, &intDate)
	if err != nil {
		log.Println("authtoken: error while validating user:", err)
		return "", time.Time{}, false
	}
	return user, time.Unix(intDate, 0), intDate > time.Now().Unix()
}

// GetAuthtokenOfUser returns all auth token associated by a user.
func GetAuthtokenOfUser(user string) ([]Authtoken, error) {
	authtoken := make([]Authtoken, 0)

	rows, err := db.Query("SELECT id,user,validUntil FROM authtoken WHERE user=?", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		a := Authtoken{}
		var intDate int64
		err = rows.Scan(&a.ID, &a.User, &intDate)
		if err != nil {
			return authtoken, err
		}
		a.ValidUntil = time.Unix(intDate, 0)
		authtoken = append(authtoken, a)
	}
	return authtoken, nil
}

func StartCleanupWorker() {
	cleanupStarted.Do(func() {
		go func() {
			for {
				now := time.Now().Unix()
				_, err := db.Exec("DELETE FROM times WHERE validUntil < ?", now)
				if err != nil {
					log.Println("authtoken: error during cleanup:", err)
				}
				time.Sleep(1 * time.Hour)
			}
		}()
	})
}

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

package database

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
)

// AddInvitation adds a new invitation from a user. It returns the ID of the new invitation.
func AddInvitation(user string) (string, error) {
	r := make([]byte, 20)
	_, err := rand.Read(r)
	if err != nil {
		return "", err
	}
	inv := base32.StdEncoding.EncodeToString(r)

	_, err = db.Exec("INSERT INTO invitations (id, creator) VALUES  (?, ?)", inv, user)
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database error:", err))
	}
	return inv, nil
}

// TestInvitation returns whether an invitation is valid.
func TestInvitation(id string) (bool, error) {
	rows, err := db.Query("SELECT COUNT(*) FROM invitations WHERE id=?", id)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var count int

	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return false, err
		}
	} else {
		return false, errors.New("Can not read invitation data")
	}

	if count != 0 {
		return true, nil
	}
	return false, nil
}

// RemoveInvitation returns the invitation (identified by ID) from the database. It is no longer valid.
func RemoveInvitation(id string) error {
	test, err := TestInvitation(id)
	if err != nil {
		return err
	}
	if !test {
		return errors.New("Invitation is not valid")
	}

	r, err := db.Exec("DELETE FROM invitations WHERE id=?", id)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return errors.New(fmt.Sprintln("Database count error:", err))
	}

	if count != 1 {
		return errors.New(fmt.Sprintln("Delete count is", count))
	}
	return nil
}

// RemoveAllInvitation removes all invitations from the database, making all invalid.
func RemoveAllInvitation() error {
	_, err := db.Exec("DELETE FROM invitations")
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// GetInvitations returns all invitations (identified by ID) created by a user.
func GetInvitations(user string) ([]string, error) {
	rows, err := db.Query("SELECT id FROM invitations WHERE creator=?", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var s string
	inv := make([]string, 0)

	for rows.Next() {
		err = rows.Scan(&s)
		if err != nil {
			return nil, err
		}
		inv = append(inv, s)
	}
	return inv, nil
}

// GetInvitationCreator returns the creator of an invitation (identified by ID).
func GetInvitationCreator(id string) (string, error) {
	test, err := TestInvitation(id)
	if err != nil {
		return "", err
	}
	if !test {
		return "", errors.New("Invitation is not valid")
	}

	rows, err := db.Query("SELECT creator FROM invitations WHERE id=?", id)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var s string

	if rows.Next() {
		err = rows.Scan(&s)
		if err != nil {
			return "", err
		}
	} else {
		return "", errors.New("Can not read invitation data")
	}
	return s, nil
}

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
	"errors"
	"fmt"
)

// DeleteUser removes a user and all associated information from the database.
// It returns the number of removed informations.
// It can not be reverted.
func DeleteUser(user string) (int64, error) {
	verify, err := UserExists(user)
	if err != nil {
		return 0, err
	}
	if !verify {
		return 0, errors.New("User not found")
	}

	defer setLastUpdateTopicPost()

	userData, err := GetUser(user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	countAll := int64(0)

	r, err := tx.Exec("DELETE FROM post WHERE poster=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}

	countAll += count

	r, err = tx.Exec("DELETE FROM topic WHERE creator=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err = r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}

	countAll += count

	r, err = tx.Exec("DELETE FROM invitations WHERE creator=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err = r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}

	countAll += count

	if userData.InvidedBy == "" {
		userData.InvidedBy = "SYSTEM: DELETED USER"
	}
	r, err = tx.Exec("UPDATE user SET invitedby=?, invitationdirect=? WHERE invitedby=?", userData.InvidedBy, false, user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err = r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}

	r, err = tx.Exec("DELETE FROM user WHERE name=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err = r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}

	countAll += count

	err = tx.Commit()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	err = nil

	return countAll, err
}

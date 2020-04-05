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
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

// calculatePW calculates a password hash out of the password and the salt of a user.
// The process is deterministic - the same password/salt combination will always return the same hash.
// The hash is calculated according to the best practice. Currently, Argon2 is used.
// It is required by the caller to provide a secure salt. The salt should be base64 encoded.
func calculatePW(pw, salt string) (string, error) {
	bsalt, err := base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), bsalt, 1, 64*1024, 2, 33)
	return base64.StdEncoding.EncodeToString(key), nil
}

// generateSalt returns a secure salt.
// The salt is generated from "ctypto/rand" and therefore should be save for usage in cryptography.
func generateSalt() (string, error) {
	salt := make([]byte, 33)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// UserExists returns whether the user is known in the database.
// It only returns an error if a communication problem with the database occured. In all other cases, the bool signals whether the user is known.
func UserExists(user string) (bool, error) {
	rows, err := db.Query("SELECT COUNT(*) FROM user WHERE name=?", user)
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
		return false, errors.New("Can not read user data")
	}

	return count != 0, nil
}

// GetUser returns a user struct with all information of the associated user (except password hash and salt).
// Returns an error if the user does not exist.
func GetUser(user string) (User, error) {
	exists, err := UserExists(user)
	if err != nil {
		return User{}, err
	}
	if !exists {
		return User{}, errors.New("User does not exist")
	}

	rows, err := db.Query("SELECT name, admin, comment, invitedby, invitationdirect, lastseen FROM user WHERE name=?", user)
	if err != nil {
		return User{}, err
	}
	defer rows.Close()

	var u User

	if rows.Next() {
		var lastSeenInt int64
		err = rows.Scan(&u.Name, &u.Admin, &u.Comment, &u.InvidedBy, &u.InvitationDirect, &lastSeenInt)
		if err != nil {
			return User{}, err
		}
		u.LastSeen = time.Unix(lastSeenInt, 0)
	} else {
		return User{}, errors.New("Can not read user data")
	}

	return u, nil
}

// GetAllUser returns all user currently known to the database.
func GetAllUser() ([]User, error) {
	rows, err := db.Query("SELECT name, admin, comment, invitedby, invitationdirect, lastseen FROM user ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)

	for rows.Next() {
		var u User
		var lastSeenInt int64
		err = rows.Scan(&u.Name, &u.Admin, &u.Comment, &u.InvidedBy, &u.InvitationDirect, &lastSeenInt)
		if err != nil {
			return nil, err
		}
		u.LastSeen = time.Unix(lastSeenInt, 0)
		users = append(users, u)
	}
	return users, nil
}

// VerifyUser returns whether the user exists and the provided password is correct.
// It should only return an error on communication problems with the database, but not if any checks fail.
func VerifyUser(user, pw string) (bool, error) {
	rows, err := db.Query("SELECT encodedpasswort, salt FROM user WHERE name=?", user)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var encodedPassword, salt string

	if rows.Next() {
		err = rows.Scan(&encodedPassword, &salt)
		if err != nil {
			return false, err
		}
	} else {
		// User does not exist
		return false, nil
	}
	key, err := calculatePW(pw, salt)
	if err != nil {
		return false, err
	}

	return encodedPassword == key, nil
}

// IsAdmin returns whether a user is administrator.
// Returns an error if the user does not exist.
func IsAdmin(user string) (bool, error) {
	exists, err := UserExists(user)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, errors.New("User does not exist")
	}

	rows, err := db.Query("SELECT admin FROM user WHERE name=?", user)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var admin bool

	if rows.Next() {
		err = rows.Scan(&admin)
		if err != nil {
			return false, err
		}
	} else {
		return false, errors.New("Can not read user data")
	}

	return admin, nil
}

// SetAdmin sets whether a user is administrator.
// Returns an error if the user does not exist.
func SetAdmin(user string, admin bool) error {
	exists, err := UserExists(user)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("User does not exist")
	}

	_, err = db.Exec("UPDATE user SET admin=? WHERE name=?", admin, user)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// SetComment updates the comment of a user.
// Returns an error if the user does not exist.
func SetComment(user string, comment string) error {
	exists, err := UserExists(user)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New("User does not exist")
	}

	_, err = db.Exec("UPDATE user SET comment=? WHERE name=?", comment, user)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// SetInvitedby sets which user invited the user.
// Returns an error if the target user does not exist. It does not check if the invitor does not exists.
// This is intentioal, allowing 'pseudo user' as an inviter.
func SetInvitedby(user, invitedby string, direct bool) error {
	exists, err := UserExists(user)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New("User does not exist")
	}

	_, err = db.Exec("UPDATE user SET invitedby=?, invitationdirect=? WHERE name=?", invitedby, direct, user)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// AddUser adds a new user to the database. Admin status is automatically set to the provided value.
// Returns an error if the user alreasy exist.
func AddUser(user, pw string, admin bool) error {
	exists, err := UserExists(user)

	if err != nil {
		return err
	}

	if exists {
		return errors.New("User already exists")
	}

	salt, err := generateSalt()
	if err != nil {
		return err
	}

	encodedPassword, err := calculatePW(pw, salt)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	_, err = db.Exec("INSERT INTO user (name, salt, encodedpasswort, admin) VALUES (?, ?, ?, ?)", user, salt, encodedPassword, admin)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// EditPassword changes the password of a user.
// Returns an error if the user does not exist.
func EditPassword(user, pw string) error {
	exists, err := UserExists(user)

	if err != nil {
		return err
	}

	if !exists {
		return errors.New("User does not exist")
	}

	salt, err := generateSalt()
	if err != nil {
		return err
	}

	encodedPassword, err := calculatePW(pw, salt)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	_, err = db.Exec("UPDATE user SET salt=?, encodedpasswort=? WHERE name=?", salt, encodedPassword, user)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// ModifyLastSeen sets the 'last seen' status of a user to the current time.
// Returns an error if the user does not exist.
func ModifyLastSeen(user string) error {
	exists, err := UserExists(user)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("User does not exist")
	}

	lastseen := time.Now().Unix()
	_, err = db.Exec("UPDATE user SET lastseen=? WHERE name=?", lastseen, user)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

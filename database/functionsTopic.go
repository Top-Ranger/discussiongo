// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2021 Marcus Soll
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
	"strconv"
	"time"
)

// GetTopics returns all topics currently saved in the database.
func GetTopics() ([]Topic, error) {
	rows, err := db.Query("SELECT * FROM topic ORDER BY lastmodified DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topics := make([]Topic, 0)

	for rows.Next() {
		t := Topic{}
		var created int64
		var modified int64
		var intID int64
		err = rows.Scan(&intID, &t.Name, &t.Creator, &created, &modified, &t.Closed, &t.Pinned)
		if err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(intID, 10)
		t.Created = time.Unix(created, 0)
		t.LastModified = time.Unix(modified, 0)
		topics = append(topics, t)
	}
	return topics, nil
}

// GetTopicsByUser returns all topics belonging to a user currently saved in the database.
func GetTopicsByUser(user string) ([]Topic, error) {
	exists, err := UserExists(user)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("User does not exist")
	}

	rows, err := db.Query("SELECT * FROM topic WHERE creator=? ORDER BY lastmodified DESC", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	topics := make([]Topic, 0)

	for rows.Next() {
		t := Topic{}
		var created int64
		var modified int64
		var intID int64
		err = rows.Scan(&intID, &t.Name, &t.Creator, &created, &modified, &t.Closed, &t.Pinned)
		if err != nil {
			return nil, err
		}
		t.ID = strconv.FormatInt(intID, 10)
		t.Created = time.Unix(created, 0)
		t.LastModified = time.Unix(modified, 0)
		topics = append(topics, t)
	}
	return topics, nil
}

// GetTopic returns the topic currently associated by the ID.
func GetTopic(ID string) (Topic, error) {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return Topic{}, errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	rows, err := db.Query("SELECT * FROM topic WHERE id=?", intID)
	if err != nil {
		return Topic{}, err
	}
	defer rows.Close()

	t := Topic{}
	if rows.Next() {
		var created int64
		var modified int64
		var intID int64
		err = rows.Scan(&intID, &t.Name, &t.Creator, &created, &modified, &t.Closed, &t.Pinned)
		if err != nil {
			return t, err
		}
		t.ID = strconv.FormatInt(intID, 10)
		t.Created = time.Unix(created, 0)
		t.LastModified = time.Unix(modified, 0)
	} else {
		return t, errors.New("Can not read topic data")
	}
	return t, nil
}

// AddTopic adds a new topic to the database.
// The modification time is set to the current time.
func AddTopic(name, creator string) (string, error) {
	defer SetLastUpdateTopicPost()
	date := time.Now().Unix()
	r, err := db.Exec("INSERT INTO topic (name, creator, created, lastmodified) VALUES (?, ?, ?, ?)", name, creator, date, date)
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database error:", err))
	}

	id, err := r.LastInsertId()
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database id error:", err))
	}

	return strconv.FormatInt(id, 10), nil
}

// TopicModifyTime sets the modification time of a topic to the current time.
func TopicModifyTime(ID string) error {
	defer SetLastUpdateTopicPost()
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}
	date := time.Now().Unix()
	r, err := db.Exec("UPDATE topic SET lastmodified=? WHERE id=?", date, intID)
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

// TopicSetClosed sets the 'closed' property of a topic to the given value.
// It does not affect the modification time.
func TopicSetClosed(ID string, closed bool) error {
	defer SetLastUpdateTopicPost()
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	r, err := db.Exec("UPDATE topic SET closed=? WHERE id=?", closed, intID)
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

// TopicSetPinned sets the 'pinned' property of a topic to the given value.
// It does not affect the modification time.
func TopicSetPinned(ID string, pinned bool) error {
	defer SetLastUpdateTopicPost()
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	r, err := db.Exec("UPDATE topic SET pinned=? WHERE id=?", pinned, intID)
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

// DeleteTopic removes a topic and all associated posts from the database.
// This action can not be undone.
func DeleteTopic(ID string) error {
	defer SetLastUpdateTopicPost()
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	tx, err := db.Begin()
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	r, err := tx.Exec("DELETE FROM topic WHERE id=?", intID)
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

	_, err = tx.Exec("DELETE FROM post WHERE topic=?", intID)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	err = tx.Commit()
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	err = nil

	return err
}

// RenameTopic renames a topic.
// It does affect the modification time.
func RenameTopic(ID string, newName string) error {
	defer SetLastUpdateTopicPost()
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	r, err := db.Exec("UPDATE topic SET name=?,lastmodified=? WHERE id=?", newName, time.Now().Unix(), intID)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return errors.New(fmt.Sprintln("Database count error:", err))
	}

	if count != 1 {
		return errors.New(fmt.Sprintln("Update count is", count))
	}
	return nil
}

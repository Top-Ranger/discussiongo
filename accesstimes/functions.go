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

package accesstimes

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// AccessTimes represents a time when a user accessed a topic.
type AccessTimes struct {
	// TopicID is the ID of the topic
	TopicID string

	// Time is the access time
	Time time.Time
}

// SaveTime saves the access time of a user / topic combination into the database.
// Since the write is done asynchronously, you have to call WaitWrite if you need to ensure that the time is actually written (and GetTimes / GetUserTimes returns the new value).
func SaveTime(user string, topic string, access time.Time) error {
	t, err := strconv.Atoi(topic)
	if err != nil {
		return err
	}
	saveTime <- save{
		Name:  user,
		Topic: t,
		Time:  access,
	}
	return nil
}

// GetTimes returns the access times of a user for the given topics.
func GetTimes(user string, topics []string) ([]time.Time, error) {
	if len(topics) == 0 {
		return make([]time.Time, 0), nil
	}

	stmt, err := db.Prepare("SELECT time FROM times WHERE name=? AND topic=?")
	if err != nil {
		return nil, err
	}

	t := make([]time.Time, len(topics))
	for i := range topics {
		id, err := strconv.Atoi(topics[i])
		if err != nil {
			return t, err
		}
		row, err := stmt.Query(user, id)
		defer row.Close()
		if err != nil {
			return t, err
		}
		var timeInt int64
		if row.Next() {
			err = row.Scan(&timeInt)
			if err != nil {
				return t, err
			}
			t[i] = time.Unix(timeInt, 0)
		}
		row.Close()
	}
	return t, nil
}

// DeleteUser removes all information of a user from the database and returns the number of removed data points.
// It does not ensure that no new data is written once the call returns.
func DeleteUser(user string) (int64, error) {
	deleteUser <- user
	r, err := db.Exec("DELETE FROM times WHERE name=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}
	deleteUser <- user
	return count, nil
}

// GetUserTimes returns all accesstimes saved by a user.
func GetUserTimes(user string) ([]AccessTimes, error) {
	rows, err := db.Query("SELECT time, topic FROM times WHERE name=?", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	t := make([]AccessTimes, 0)
	for rows.Next() {
		var topicInt int
		var timeInt int64
		err = rows.Scan(&timeInt, &topicInt)
		if err != nil {
			return t, err
		}
		t = append(t, AccessTimes{TopicID: strconv.Itoa(topicInt), Time: time.Unix(timeInt, 0)})
	}
	return t, nil
}

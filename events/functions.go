// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Marcus Soll
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

package events

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

const AnoymousUser = "SYSTEM: DELETED USER"

// Event represents a file.
type Event struct {
	ID    string
	Type  int
	User  string
	Topic string
	Date  time.Time
	Data  []byte `xml:",cdata"`
}

// DeleteTopicEvents removes all events associated by a topic.
// It returns the number of deleted events.
func DeleteTopicEvents(topicid string) (int64, error) {
	r, err := db.Exec("DELETE FROM events WHERE topic=?", topicid)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}
	return count, nil
}

// AnonymiseUserEvents removes the user name from all events associated by a user.
// It returns the number of renamed events.
func AnonymiseUserEvents(user string) (int64, error) {
	r, err := db.Exec("UPDATE events SET user=? WHERE user=?", AnoymousUser, user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}
	return count, nil
}

// DeleteEvent removes a single event.
func DeleteEvent(ID string) error {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	_, err = db.Exec("DELETE FROM events WHERE id=?", intID)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// SaveEvent saves an event.
// ID and time will be ignored.
// It returns the ID of the event.
func SaveEvent(e Event) (string, error) {
	date := time.Now().Unix()
	r, err := db.Exec("INSERT INTO events (type, user, topic, date, data) VALUES (?, ?, ?, ?, ?)", e.Type, e.User, e.Topic, date, e.Data)
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database error:", err))
	}

	id, err := r.LastInsertId()
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database id error:", err))
	}

	return strconv.FormatInt(id, 10), nil

}

// GetEvent returns a file by ID.
func GetEvent(ID string) (Event, error) {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return Event{}, errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	rows, err := db.Query("SELECT * FROM events WHERE id=?", intID)
	if err != nil {
		return Event{}, err
	}
	defer rows.Close()

	e := Event{}
	if rows.Next() {
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &e.Type, &e.User, &e.Topic, &intDate, &e.Data)
		if err != nil {
			return e, err
		}
		e.ID = strconv.FormatInt(intID, 10)
		e.Date = time.Unix(intDate, 0)
	} else {
		return e, errors.New("Can not read topic data")
	}
	return e, nil
}

// GetEventsOfTopic returns all events associated by a topic.
func GetEventsOfTopic(topicid string) ([]Event, error) {
	events := make([]Event, 0)

	rows, err := db.Query("SELECT * FROM events WHERE topic=?", topicid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Event{}
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &e.Type, &e.User, &e.Topic, &intDate, &e.Data)
		if err != nil {
			return events, err
		}
		e.ID = strconv.FormatInt(intID, 10)
		e.Date = time.Unix(intDate, 0)
		events = append(events, e)
	}
	return events, nil
}

// GetEventsOfUser returns all events associated by a user.
func GetEventsOfUser(user string) ([]Event, error) {
	events := make([]Event, 0)

	rows, err := db.Query("SELECT * FROM events WHERE user=?", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		e := Event{}
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &e.Type, &e.User, &e.Topic, &intDate, &e.Data)
		if err != nil {
			return events, err
		}
		e.ID = strconv.FormatInt(intID, 10)
		e.Date = time.Unix(intDate, 0)
		events = append(events, e)
	}
	return events, nil
}

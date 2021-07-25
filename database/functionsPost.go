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

// GetPosts returns all posts of a topic from the database.
func GetPosts(topicID string) ([]Post, error) {
	topicIntID, err := strconv.ParseInt(topicID, 10, 64)
	if err != nil {
		return nil, errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	rows, err := db.Query("SELECT * FROM post WHERE topic=? ORDER BY time ASC", topicIntID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)

	for rows.Next() {
		p := Post{}
		var timeInt int64
		var topicInt int64
		var intID int64
		err = rows.Scan(&intID, &p.Content, &p.Poster, &timeInt, &topicInt)
		if err != nil {
			return nil, err
		}
		p.ID = strconv.FormatInt(intID, 10)
		p.TopicID = strconv.FormatInt(topicInt, 10)
		p.Time = time.Unix(timeInt, 0)
		posts = append(posts, p)
	}
	return posts, nil
}

// GetSinglePost returns the post associated with the given ID.
func GetSinglePost(ID string) (Post, error) {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return Post{}, errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	rows, err := db.Query("SELECT * FROM post WHERE id=?", intID)
	if err != nil {
		return Post{}, err
	}
	defer rows.Close()

	if rows.Next() {
		p := Post{}
		var timeInt int64
		var topicInt int64
		var intID int64
		err = rows.Scan(&intID, &p.Content, &p.Poster, &timeInt, &topicInt)
		if err != nil {
			return Post{}, err
		}
		p.ID = strconv.FormatInt(intID, 10)
		p.TopicID = strconv.FormatInt(topicInt, 10)
		p.Time = time.Unix(timeInt, 0)
		return p, nil
	}
	return Post{}, errors.New("No such post")
}

// GetPostsByUser returns all posts of a user from the database.
func GetPostsByUser(user string) ([]Post, error) {
	exists, err := UserExists(user)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("User does not exist")
	}

	rows, err := db.Query("SELECT * FROM post WHERE poster=? ORDER BY time DESC", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)

	for rows.Next() {
		p := Post{}
		var timeInt int64
		var topicInt int64
		var intID int64
		err = rows.Scan(&intID, &p.Content, &p.Poster, &timeInt, &topicInt)
		if err != nil {
			return nil, err
		}
		p.ID = strconv.FormatInt(intID, 10)
		p.TopicID = strconv.FormatInt(topicInt, 10)
		p.Time = time.Unix(timeInt, 0)
		posts = append(posts, p)
	}
	return posts, nil
}

// AddPost saves a post to the database.
func AddPost(topicID, user, content string) (string, error) {
	defer SetLastUpdateTopicPost()
	topicIntID, err := strconv.ParseInt(topicID, 10, 64)
	if err != nil {
		return "", errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	date := time.Now().Unix()
	r, err := db.Exec("INSERT INTO post (content, poster, time, topic) VALUES (?, ?, ?, ?)", content, user, date, topicIntID)
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database error:", err))
	}

	id, err := r.LastInsertId()
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database id error:", err))
	}

	return strconv.FormatInt(id, 10), nil
}

// DeletePost removes a post completely from the database. This action can not be undone.
func DeletePost(topicID, ID string) error {
	defer SetLastUpdateTopicPost()
	topicIntID, err := strconv.ParseInt(topicID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}
	postIntID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	r, err := db.Exec("DELETE FROM post WHERE id=? AND topic=?", postIntID, topicIntID)
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

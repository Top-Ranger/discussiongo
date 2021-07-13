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

package files

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// File represents a file.
type File struct {
	ID    string
	Name  string
	User  string
	Topic string
	Date  time.Time
	Data  []byte
}

// DeleteTopicFiles removes all files associated by a topic.
// It returns the number of deleted files.
func DeleteTopicFiles(topicid int) (int64, error) {
	r, err := db.Exec("DELETE FROM files WHERE topic=?", topicid)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}
	return count, nil
}

// DeleteUserFiles removes all files associated by a user.
// It returns the number of deleted files.
func DeleteUserFiles(user string) (int64, error) {
	r, err := db.Exec("DELETE FROM files WHERE user=?", user)
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database error:", err))
	}

	count, err := r.RowsAffected()
	if err != nil {
		return 0, errors.New(fmt.Sprintln("Database count error:", err))
	}
	return count, nil
}

// DeleteFile removes a single file.
func DeleteFile(ID string) error {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	_, err = db.Exec("DELETE FROM files WHERE id=?", intID)
	if err != nil {
		return errors.New(fmt.Sprintln("Database error:", err))
	}

	return nil
}

// SaveFile saves a file.
// ID and time will be ignored.
// It returns the ID of the file.
func SaveFile(f File) (string, error) {
	date := time.Now().Unix()
	r, err := db.Exec("INSERT INTO files (name, user, topic, date, data) VALUES (?, ?, ?, ?, ?)", f.Name, f.User, f.Topic, date, f.Data)
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database error:", err))
	}

	id, err := r.LastInsertId()
	if err != nil {
		return "", errors.New(fmt.Sprintln("Database id error:", err))
	}

	return strconv.FormatInt(id, 10), nil

}

// GetFile returns a file by ID.
func GetFile(ID string) (File, error) {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return File{}, errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	rows, err := db.Query("SELECT * FROM files WHERE id=?", intID)
	if err != nil {
		return File{}, err
	}
	defer rows.Close()

	f := File{}
	if rows.Next() {
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &f.Name, &f.User, &f.Topic, &intDate, &f.Data)
		if err != nil {
			return f, err
		}
		f.ID = strconv.FormatInt(intID, 10)
		f.Date = time.Unix(intDate, 0)
	} else {
		return f, errors.New("Can not read topic data")
	}
	return f, nil
}

// GetFile returns the file metadata by ID.
// It does not fill File.Data
func GetFileMetadata(ID string) (File, error) {
	intID, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		return File{}, errors.New(fmt.Sprintln("Can not convert ID:", err))
	}

	rows, err := db.Query("SELECT id,name,user,topic,date FROM files WHERE id=?", intID)
	if err != nil {
		return File{}, err
	}
	defer rows.Close()

	f := File{}
	if rows.Next() {
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &f.Name, &f.User, &f.Topic, &intDate)
		if err != nil {
			return f, err
		}
		f.ID = strconv.FormatInt(intID, 10)
		f.Date = time.Unix(intDate, 0)
	} else {
		return f, errors.New("Can not read topic data")
	}
	return f, nil
}

// GetFilesOfTopic returns all file metadata associated by a topic.
// It does not fill File.Data
func GetFileMetadataOfTopic(topicid string) ([]File, error) {
	files := make([]File, 0)

	rows, err := db.Query("SELECT id,name,user,topic,date FROM files WHERE topic=?", topicid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		f := File{}
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &f.Name, &f.User, &f.Topic, &intDate)
		if err != nil {
			return files, err
		}
		f.ID = strconv.FormatInt(intID, 10)
		f.Date = time.Unix(intDate, 0)
		files = append(files, f)
	}
	return files, nil
}

// GetFilesForUser returns all files associated by a user.
func GetFilesForUser(user string) ([]File, error) {
	files := make([]File, 0)

	rows, err := db.Query("SELECT * FROM files WHERE user=?", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		f := File{}
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &f.Name, &f.User, &f.Topic, &intDate, &f.Data)
		if err != nil {
			return files, err
		}
		f.ID = strconv.FormatInt(intID, 10)
		f.Date = time.Unix(intDate, 0)
		files = append(files, f)
	}
	return files, nil
}

// GetFilesForUser returns all file metadata associated by a user.
// It does not fill File.Data
func GetFileMetadataForUser(user string) ([]File, error) {
	files := make([]File, 0)

	rows, err := db.Query("SELECT id,name,user,topic,date FROM files WHERE user=?", user)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		f := File{}
		var intDate int64
		var intID int64
		err = rows.Scan(&intID, &f.Name, &f.User, &f.Topic, &intDate)
		if err != nil {
			return files, err
		}
		f.ID = strconv.FormatInt(intID, 10)
		f.Date = time.Unix(intDate, 0)
		files = append(files, f)
	}
	return files, nil
}

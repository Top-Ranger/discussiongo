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

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/files"
)

func init() {
	http.HandleFunc("/postFile.html", saveFileHandleFunc)
	http.HandleFunc("/getFile.html", getFileHandleFunc)
	http.HandleFunc("/deleteFile.html", deleteFileHandleFunc)
}

func saveFileHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	isAdmin := false
	if loggedIn {
		var err error
		isAdmin, err = database.IsAdmin(user)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		SetCookies(rw, user)
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	canUpload := config.EnableFileUpload || (config.EnableFileUploadAdmin && isAdmin)

	if !canUpload {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err := r.ParseMultipartForm(int64(config.FileMaxMB) * 1000000)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	token := r.Form.Get("token")

	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic := r.Form.Get("topic")

	topicData, err := database.GetTopic(topic)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	if topicData.Closed {
		tl := GetDefaultTranslation()
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(tl.Closed))
		return

	}

	fileReader, meta, err := r.FormFile("file")
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	b, err := io.ReadAll(fileReader)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	f := files.File{
		Name:  meta.Filename,
		User:  user,
		Topic: topic,
		Data:  b,
	}

	_, err = files.SaveFile(f)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/topic.html?id=%s#files", config.ServerPath, topic), http.StatusFound)
}

func getFileHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !config.CanReadWithoutRegister && !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	if loggedIn {
		var err error
		_, err = database.IsAdmin(user)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		SetCookies(rw, user)
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	q := r.URL.Query()
	id, ok := q["id"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(id) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	f, err := files.GetFile(id[0])

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(f.Data)
}

func deleteFileHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	isAdmin, err := database.IsAdmin(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.URL.Query()
	id, ok := q["id"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(id) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	token, ok := q["token"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(token) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	f, err := files.GetFileMetadata(id[0])
	if err != nil {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	if !isAdmin && user != f.User {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	err = files.DeleteFile(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
}
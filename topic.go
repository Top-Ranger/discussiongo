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

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
)

type templateTopicData struct {
	ServerPath    string
	ForumName     string
	LoggedIn      bool
	User          string
	IsAdmin       bool
	HasPinned     bool
	HasClosed     bool
	HasNew        bool
	CurrentUpdate int64
	Topics        []topicData
	TopicsPinned  []topicData
	TopicsClosed  []topicData
	Token         string
	Translation   Translation
}

type topicData struct {
	ID       string
	Name     string
	Modified string
	Creator  string
	Closed   bool
	Pinned   bool
	New      bool
}

var (
	topicTemplate *template.Template
)

func init() {
	funcMap := template.FuncMap{
		"even": func(i int) bool {
			return i%2 == 0
		},
	}

	b, err := os.ReadFile("template/topics.html")
	if err != nil {
		panic(err)
	}
	topicTemplate, err = template.New("topic").Funcs(funcMap).Parse(string(b))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", topicHandleFunc)
	http.HandleFunc("/newTopic.html", newTopicHandleFunc)
	http.HandleFunc("/deleteTopic.html", deleteTopicHandleFunc)
	http.HandleFunc("/closeTopic.html", closeTopicHandleFunc)
	http.HandleFunc("/pinTopic.html", pinTopicHandleFunc)
}

func topicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !config.CanReadWithoutRegister && !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
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

	topics, err := database.GetTopics()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	td := templateTopicData{
		ServerPath:    config.ServerPath,
		ForumName:     config.ForumName,
		LoggedIn:      loggedIn,
		User:          user,
		IsAdmin:       isAdmin,
		HasPinned:     false,
		HasClosed:     false,
		HasNew:        false,
		CurrentUpdate: database.GetLastUpdateTopicPost(),
		Topics:        make([]topicData, 0, len(topics)),
		TopicsPinned:  make([]topicData, 0, len(topics)),
		TopicsClosed:  make([]topicData, 0, len(topics)),
		Translation:   GetDefaultTranslation(),
	}

	var times []time.Time

	if loggedIn {
		var err error
		token, err := data.GetStringsTimed(time.Now(), fmt.Sprintf("%s;Token", user))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		td.Token = token

		ids := make([]string, len(topics))
		for i := range topics {
			ids[i] = topics[i].ID
		}
		times, err = accesstimes.GetTimes(user, ids)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
	}

	for i := range topics {
		t := topicData{
			ID:       topics[i].ID,
			Name:     topics[i].Name,
			Modified: topics[i].LastModified.Format(time.RFC822),
			Creator:  topics[i].Creator,
			Closed:   topics[i].Closed,
			Pinned:   topics[i].Pinned,
			New:      false,
		}

		if loggedIn {
			if times[i].Before(topics[i].LastModified) {
				t.New = true
				td.HasNew = true
			}
		}

		if t.Closed {
			td.TopicsClosed = append(td.TopicsClosed, t)
			td.HasClosed = true
		} else if t.Pinned {
			td.TopicsPinned = append(td.TopicsPinned, t)
			td.HasPinned = true
		} else {
			td.Topics = append(td.Topics, t)
		}
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = topicTemplate.Execute(rw, td)
	if err != nil {
		log.Println("Error executing topic template:", err)
	}
}

func newTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.Form

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

	topic, ok := q["topic"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(topic) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(strings.TrimSpace(topic[0])) == 0 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	id, err := database.AddTopic(topic[0], user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/topic.html?id=%s", config.ServerPath, id), http.StatusFound)
}

func deleteTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
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
	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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

	err = database.DeleteTopic(id[0])
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

func closeTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
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

	closed, ok := q["closed"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(closed) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if closed[0] != "0" && closed[0] != "1" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic, err := database.GetTopic(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !isAdmin && user != topic.Creator {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err = database.TopicSetClosed(id[0], closed[0] == "1")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/#topic%s", config.ServerPath, id[0]), http.StatusFound)
}

func pinTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
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
	if !isAdmin {
		rw.WriteHeader(http.StatusForbidden)
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

	pin, ok := q["pin"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(pin) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if pin[0] != "0" && pin[0] != "1" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	// Test if topic exists
	_, err = database.GetTopic(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.TopicSetPinned(id[0], pin[0] == "1")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/#topic%s", config.ServerPath, id[0]), http.StatusFound)
}

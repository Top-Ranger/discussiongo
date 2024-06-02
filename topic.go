// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2021,2022,2024 Marcus Soll
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
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
	"github.com/Top-Ranger/discussiongo/files"
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
	var err error

	topicTemplate, err = template.New("topic").Funcs(evenOddFuncMap).ParseFS(templateFiles, "template/topics.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", topicHandleFunc)
	http.HandleFunc("/newTopic.html", newTopicHandleFunc)
	http.HandleFunc("/deleteTopic.html", deleteTopicHandleFunc)
	http.HandleFunc("/closeTopic.html", closeTopicHandleFunc)
	http.HandleFunc("/pinTopic.html", pinTopicHandleFunc)
	http.HandleFunc("/renameTopic.html", renameTopicHandleFunc)
}

func topicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

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

	err = topicTemplate.ExecuteTemplate(rw, "topics.html", td)
	if err != nil {
		log.Println("Error executing topic template:", err)
	}
}

func newTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

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

	token := q.Get("token")
	if token == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic := q.Get("topic")
	if len(strings.TrimSpace(topic)) == 0 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	id, err := database.AddTopic(topic, user)
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
	loggedIn, user := TestUser(r, rw)

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
	id := q.Get("id")
	if id == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	token := q.Get("token")
	if token == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic, err := database.GetTopic(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	_, err = files.DeleteTopicFiles(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	_, err = events.DeleteTopicEvents(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.DeleteTopic(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	deletionEvent := events.Event{
		Type:  EventTopicDeleted,
		User:  user,
		Topic: eventAdminPseudoTopic,
		Date:  time.Now(),
		Data:  []byte(topic.Name),
	}

	_, err = events.SaveEvent(deletionEvent)
	if err != nil {
		log.Printf("Can not save event %+v: %s", deletionEvent, err.Error())
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
}

func closeTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

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
	id := q.Get("id")
	if id == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	token := q.Get("token")
	if token == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	closed := q.Get("closed")
	if closed != "0" && closed != "1" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic, err := database.GetTopic(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !config.EveryoneCanCloseAndOpenTopics && !isAdmin && user != topic.Creator {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err = database.TopicSetClosed(id, closed == "1")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	event := events.Event{
		Type:  EventOpenTopic,
		User:  user,
		Topic: id,
		Date:  time.Now(),
	}

	if closed == "1" {
		event.Type = EventCloseTopic
	}

	_, err = events.SaveEvent(event)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/#topic%s", config.ServerPath, id), http.StatusFound)
}

func pinTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

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
	id := q.Get("id")
	if id == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	token := q.Get("token")
	if token == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	pin := q.Get("pin")
	if pin != "0" && pin != "1" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	// Test if topic exists
	_, err = database.GetTopic(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.TopicSetPinned(id, pin == "1")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	event := events.Event{
		Type:  EventUnpinTopic,
		User:  user,
		Topic: id,
		Date:  time.Now(),
	}

	if pin == "1" {
		event.Type = EventPinTopic
	}

	_, err = events.SaveEvent(event)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/#topic%s", config.ServerPath, id), http.StatusFound)
}

func renameTopicHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

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

	err = r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.Form

	token := q.Get("token")
	if token == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	id := q.Get("id")
	if id == "" {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	newtopic := q.Get("newtopic")
	if len(strings.TrimSpace(newtopic)) == 0 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic, err := database.GetTopic(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !isAdmin && user != topic.Creator {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err = database.RenameTopic(id, newtopic)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	e := events.Event{
		Type:  EventTopicRenamed,
		User:  user,
		Topic: id,
		Date:  time.Now(),
		Data:  eventCreateTopicRenameData(topic.Name, newtopic),
	}
	_, err = events.SaveEvent(e)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	e.Topic = eventAdminPseudoTopic
	_, err = events.SaveEvent(e)
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

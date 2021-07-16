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
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
)

func init() {
	http.HandleFunc("/deleteEvent.html", deleteEventHandleFunc)
}

const (
	EventCloseTopic = iota
	EventOpenTopic
	EventPinTopic
	EventUnpinTopic
	EventTopicRenamed
	EventPostDeleted
	EventFileDeleted
)

type eventData struct {
	ID          string
	Description template.HTML
	User        string
	Topic       string
	Date        string
	New         bool
	RealUser    bool
}

func deleteEventHandleFunc(rw http.ResponseWriter, r *http.Request) {
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

	tid := q["tid"]

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

	err = events.DeleteEvent(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	if len(tid) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	http.Redirect(rw, r, fmt.Sprintf("%s/topic.html?id=%s", config.ServerPath, url.QueryEscape(tid[0])), http.StatusFound)
}

func eventToEventData(e events.Event) eventData {
	// No new is set
	tl := GetDefaultTranslation()
	ed := eventData{
		ID:       e.ID,
		User:     e.User,
		Topic:    e.Topic,
		Date:     e.Date.Format(time.RFC822),
		RealUser: e.User != events.AnoymousUser,
	}
	switch e.Type {
	case EventCloseTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventCloseTopic))
	case EventOpenTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventOpenTopic))
	case EventPinTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventPinTopic))
	case EventUnpinTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventUnpitTopic))
	case EventTopicRenamed:
		if e.Data != nil {
			split := strings.Split(string(e.Data), "﷐")
			if len(split) == 2 {
				ed.Description = template.HTML(fmt.Sprintf("%s (<i>%s</i> 🡆 <i>%s</i>)", tl.EventTopicRenamed, split[0], split[1]))
			}
		}
	case EventPostDeleted:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventPostDeleted))
	case EventFileDeleted:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventFileDeleted))
	default:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.UnknownEvent))
	}
	return ed
}

func eventCreateTopicRenameData(old, new string) []byte {
	// ﷐
	old = strings.ReplaceAll(old, "﷐", "")
	new = strings.ReplaceAll(new, "﷐", "")
	s := fmt.Sprintf("%s﷐%s", old, new)
	return []byte(s)
}

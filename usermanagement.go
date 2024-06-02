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
	"crypto/rand"
	"encoding/base64"
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

var (
	usermanagementTemplate *template.Template
)

type usermanagementTemplateData struct {
	ServerPath  string
	ForumName   string
	Username    string
	User        []userManagementStruct
	Events      []eventData
	Token       string
	Translation Translation
}

type userManagementStruct struct {
	Name               string
	Admin              bool
	Invited            bool
	InvitedBy          string
	InvitationIndirect bool
	LastSeen           string
}

func init() {
	var err error

	usermanagementTemplate, err = template.New("usermanagement").Funcs(evenOddFuncMap).ParseFS(templateFiles, "template/usermanagement.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/usermanagement.html", usermanagementHandleFunc)
	http.HandleFunc("/setAdmin.html", usermanagementSetAdminHandleFunc)
	http.HandleFunc("/adminResetPasswort.html", usermanagementAdminResetPasswortHandleFunc)
	http.HandleFunc("/adminRegisterUser.html", usermanagementAdminRegisterUserHandleFunc)
	http.HandleFunc("/adminDeleteUser.html", usermanagementAdminDeleteUserHandleFunc)
	http.HandleFunc("/adminDeleteAllInvitations.html", usermanagementAdminDeleteAllInvitationsHandleFunc)
}

func usermanagementHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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
	}

	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	userlist, err := database.GetAllUser()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	eventlist, err := events.GetEventsOfTopic(eventAdminPseudoTopic)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	token, err := data.GetStringsTimed(time.Now(), fmt.Sprintf("%s;Token", user))
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	td := usermanagementTemplateData{
		ServerPath:  config.ServerPath,
		ForumName:   config.ForumName,
		Username:    user,
		User:        make([]userManagementStruct, 0, len(userlist)),
		Events:      make([]eventData, 0, len(eventlist)),
		Token:       token,
		Translation: GetDefaultTranslation(),
	}

	for i := range userlist {
		td.User = append(td.User, userManagementStruct{
			Name:               userlist[i].Name,
			Admin:              userlist[i].Admin,
			Invited:            userlist[i].InvidedBy != "",
			InvitedBy:          userlist[i].InvidedBy,
			InvitationIndirect: !userlist[i].InvitationDirect,
			LastSeen:           userlist[i].LastSeen.Format(time.RFC822),
		})
	}

	for i := range eventlist {
		td.Events = append(td.Events, eventToEventData(eventlist[i]))
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = usermanagementTemplate.ExecuteTemplate(rw, "usermanagement.html", td)
	if err != nil {
		log.Println("Error executing user management template:", err)
	}
}

func usermanagementSetAdminHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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
	}

	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	q := r.URL.Query()
	token := q.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	name := q.Get("name")
	if name == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	admin := q.Get("admin")
	if admin != "0" && admin != "1" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	err := database.SetAdmin(name, admin == "1")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	e := events.Event{
		Type:         EventRemoveAdministrator,
		User:         name,
		AffectedUser: user,
		Topic:        eventAdminPseudoTopic,
		Date:         time.Now(),
	}

	if admin == "1" {
		e.Type = EventSetAdministrator
	}

	_, err = events.SaveEvent(e)
	if err != nil {
		log.Printf("Can not save event %+v: %s", e, err.Error())
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/usermanagement.html#user%s", config.ServerPath, name), http.StatusFound)
}

func usermanagementAdminResetPasswortHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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
	}

	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	q := r.URL.Query()
	token := q.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	name := q.Get("name")
	if name == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	b := make([]byte, 18)
	_, err := rand.Read(b)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	newPW := base64.StdEncoding.EncodeToString(b)

	err = database.EditPassword(name, newPW)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	rw.Write([]byte(fmt.Sprintf("%s: %s\n%s: %s\n%s%s/usermanagement.html#user%s", t.User, name, t.Password, newPW, config.ServerPrefix, config.ServerPath, name)))
}

func usermanagementAdminRegisterUserHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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
	}

	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	err := r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.PostForm
	token := q.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	name := q.Get("name")
	if len(strings.TrimSpace(name)) == 0 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	if protectedUserRegexp.Match([]byte(name)) {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.NameInvalid))
		return
	}

	verify, err := database.UserExists(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if verify {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.UserExists))
		return
	}

	pw := q.Get("pw")
	if len(pw) < config.LengthPassword {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(fmt.Sprintf(t.PasswortTooShort, config.LengthPassword)))
		return
	}

	err = database.AddUser(name, pw, false)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	e := events.Event{
		Type:         EventUserRegisteredByAdmin,
		User:         name,
		AffectedUser: user,
		Topic:        eventAdminPseudoTopic,
		Date:         time.Now(),
	}

	_, err = events.SaveEvent(e)
	if err != nil {
		log.Printf("Can not save event %+v: %s", e, err.Error())
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/usermanagement.html#user%s", config.ServerPath, name), http.StatusFound)
}

func usermanagementAdminDeleteUserHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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
	}

	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	q := r.URL.Query()
	token := q.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	name := q.Get("name")
	if name == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	// Needed for deletion later
	topics, err := database.GetTopicsByUser(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	posts, err := database.GetPostsByUser(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	userfiles, err := files.GetFilesForUser(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	// Add events
	// Some might be anonymised or deleted later - that is ok
	e := make([]events.Event, 0, len(posts)+len(userfiles))

	for i := range posts {
		e = append(e, events.Event{
			Type:  EventPostDeleted,
			User:  name,
			Topic: posts[i].TopicID,
			Date:  posts[i].Time,
		})
	}

	for i := range userfiles {
		e = append(e, events.Event{
			Type:  EventFileDeleted,
			User:  name,
			Topic: userfiles[i].Topic,
			Date:  userfiles[i].Date,
		})
	}

	err = events.SaveEvents(e)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	// Now delete user
	count, err := database.DeleteUser(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	c, err := accesstimes.DeleteUser(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	c, err = files.DeleteUserFiles(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	c, err = events.AnonymiseUserEvents(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	for i := range topics {
		c, err = files.DeleteTopicFiles(topics[i].ID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		count += c

		c, err = events.DeleteTopicEvents(topics[i].ID)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		count += c
	}

	deletionEvent := events.Event{
		Type:         EventUserAdminDeleted,
		User:         name,
		AffectedUser: user,
		Topic:        eventAdminPseudoTopic,
		Date:         time.Now(),
	}

	_, err = events.SaveEvent(deletionEvent)
	if err != nil {
		log.Printf("Can not save event %+v: %s", deletionEvent, err.Error())
	}

	rw.Write([]byte(fmt.Sprintf("%s: %s\n%s: %d\n", t.User, name, t.Deleted, count)))
}

func usermanagementAdminDeleteAllInvitationsHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
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
	}

	if !isAdmin {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	q := r.URL.Query()
	token := q.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	err := database.RemoveAllInvitation()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	http.Redirect(rw, r, fmt.Sprintf("%s/usermanagement.html#inv", config.ServerPath), http.StatusFound)
}

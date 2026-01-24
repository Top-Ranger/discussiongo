// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2021,2022,2024,2026 Marcus Soll
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
	"regexp"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/authtoken"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
	"github.com/Top-Ranger/discussiongo/files"
)

type templateUserData struct {
	ServerPath              string
	ForumName               string
	User                    string
	Comment                 template.HTML
	CommentUnescaped        string
	HasComment              bool
	IsAdmin                 bool
	CanInvite               bool
	LastSeen                string
	Invitations             []string
	ServerPrefix            string
	CreateInvitationMessage string
	Token                   string
	Translation             Translation
}

var (
	userTemplate        *template.Template
	protectedUserRegexp = regexp.MustCompile("S\\s*Y\\s*S\\s*T\\s*E\\s*M")
)

func init() {
	var err error

	userTemplate, err = template.ParseFS(templateFiles, "template/user.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/user.html", userHandleFunc)
	http.HandleFunc("/password.html", userChangePasswordHandleFunc)
	http.HandleFunc("/comment.html", userChangeCommentHandleFunc)
	http.HandleFunc("/newInvitation.html", userAddInvitationHandleFunc)
	http.HandleFunc("/deleteInvitation.html", userDeleteInvitationHandleFunc)
	http.HandleFunc("/deleteUser.html", userDeleteUserHandleFunc)
	http.HandleFunc("/markRead.html", userMarkReadHandleFunc)
}

func userHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	u, err := database.GetUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	inv, err := database.GetInvitations(user)
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

	td := templateUserData{
		ServerPath:              config.ServerPath,
		ForumName:               config.ForumName,
		User:                    user,
		Comment:                 formatPost(u.Comment),
		CommentUnescaped:        u.Comment,
		HasComment:              u.Comment != "",
		IsAdmin:                 u.Admin,
		CanInvite:               (config.InvitationUser) || (config.InvitationAdmin && u.Admin),
		LastSeen:                u.LastSeen.Format(time.RFC822),
		Invitations:             inv,
		ServerPrefix:            config.ServerPrefix,
		CreateInvitationMessage: config.CreateInvitationMessage,
		Token:                   token,
		Translation:             GetDefaultTranslation(),
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = userTemplate.Execute(rw, td)
	if err != nil {
		log.Println("Error executing user template:", err)
	}
}

func userChangeCommentHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
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

	comment := q.Get("comment") // Empty comment is allowed

	err = database.SetComment(user, comment)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/user.html#comment", config.ServerPath), http.StatusFound)
}

func userAddInvitationHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
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

	isAdmin := false
	isAdmin, err = database.IsAdmin(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if (config.InvitationUser) || (config.InvitationAdmin && isAdmin) {
		_, err = database.AddInvitation(user)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		http.Redirect(rw, r, fmt.Sprintf("%s/user.html#inv", config.ServerPath), http.StatusFound)
		return
	}
	rw.WriteHeader(http.StatusForbidden)
}

func userDeleteInvitationHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
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

	id := q.Get("id")
	if id == "" {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	test, err := database.TestInvitation(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !test {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	u, err := database.GetInvitationCreator(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	if u != user {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	err = database.RemoveInvitation(id)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	http.Redirect(rw, r, fmt.Sprintf("%s/user.html#inv", config.ServerPath), http.StatusFound)
}

func userChangePasswordHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
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

	old := q.Get("old")
	if old == "" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	new := q.Get("new")
	if len(new) < config.LengthPassword {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(fmt.Sprintf(t.PasswortTooShort, config.LengthPassword)))
		return
	}

	ok, err := database.VerifyUser(user, old)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.OldPasswordWrong))
		return
	}

	err = database.EditPassword(user, new)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	_, err = authtoken.DeleteUserToken(user)
	if err != nil {
		log.Printf("Can not delete auth tokens for '%s' after password change: %s", user, err.Error())
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/user.html", config.ServerPath), http.StatusFound)
}

func userDeleteUserHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
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

	name := q.Get("user")
	if name != user {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(t.InvalidRequest))
		return
	}

	// Needed for deletion later
	topics, err := database.GetTopicsByUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	posts, err := database.GetPostsByUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	userfiles, err := files.GetFilesForUser(user)
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
			User:  user,
			Topic: posts[i].TopicID,
			Date:  posts[i].Time,
		})
	}

	for i := range userfiles {
		e = append(e, events.Event{
			Type:  EventFileDeleted,
			User:  user,
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
	count, err := database.DeleteUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	c, err := accesstimes.DeleteUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	c, err = files.DeleteUserFiles(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	c, err = events.AnonymiseUserEvents(user)
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
		Type:  EventUserDeleted,
		User:  user,
		Topic: eventAdminPseudoTopic,
		Date:  time.Now(),
	}

	c, err = authtoken.DeleteUserToken(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	_, err = events.SaveEvent(deletionEvent)
	if err != nil {
		log.Printf("Can not save event %+v: %s", deletionEvent, err.Error())
	}

	RemoveCookies(r, rw)
	rw.Write([]byte(fmt.Sprintf("%s: %s\n%s: %d\n", t.User, name, t.Deleted, count)))
}

func userMarkReadHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r, rw)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	exists, err := database.UserExists(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !exists {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
	}

	topics, err := database.GetTopics()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	now := time.Now()

	for i := range topics {
		accesstimes.SaveTime(user, topics[i].ID, now)
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	accesstimes.WaitWrite()

	http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
}

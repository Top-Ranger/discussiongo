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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
)

var (
	usermanagementTemplate *template.Template
)

type usermanagementTemplateData struct {
	ServerPath string
	ForumName  string
	Username   string
	User       []userManagementStruct
	Token      string
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
	funcMap := template.FuncMap{
		"even": func(i int) bool {
			return i%2 == 0
		},
	}

	b, err := ioutil.ReadFile("template/usermanagement.html")
	if err != nil {
		panic(err)
	}
	usermanagementTemplate, err = template.New("usermanagement").Funcs(funcMap).Parse(string(b))
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
	loggedIn, user := TestUser(r)

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

	token, err := data.GetStringsTimed(time.Now(), fmt.Sprintf("%s;Token", user))
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	td := usermanagementTemplateData{
		ServerPath: config.ServerPath,
		ForumName:  config.ForumName,
		Username:   user,
		User:       make([]userManagementStruct, 0, len(userlist)),
		Token:      token,
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

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = usermanagementTemplate.Execute(rw, td)
	if err != nil {
		log.Println("Error executing user management template:", err)
	}
}

func usermanagementSetAdminHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

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
	token, ok := q["token"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}

	name, ok := q["name"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}
	if len(name) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}

	admin, ok := q["admin"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Admin wrong"))
		return
	}
	if len(admin) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Admin wrong"))
		return
	}
	if admin[0] != "0" && admin[0] != "1" {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Admin ist falsch. Muss 0 oder 1 sein."))
		return
	}

	err := database.SetAdmin(name[0], admin[0] == "1")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	http.Redirect(rw, r, fmt.Sprintf("%s/usermanagement.html#user%s", config.ServerPath, name[0]), http.StatusFound)
}

func usermanagementAdminResetPasswortHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

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
	token, ok := q["token"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}

	name, ok := q["name"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}
	if len(name) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
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

	err = database.EditPassword(name[0], newPW)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	rw.Write([]byte(fmt.Sprintf("Username: %s\nPasswort: %s\n%s/usermanagement.html#user%s", name[0], newPW, config.ServerPrefix, name[0])))
}

func usermanagementAdminRegisterUserHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

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
	token, ok := q["token"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}

	name, ok := q["name"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}
	if len(name) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}
	if len(strings.TrimSpace(name[0])) == 0 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}

	if protectedUserRegexp.Match([]byte(name[0])) {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte("Benutzername nicht erlaubt"))
		return
	}

	verify, err := database.UserExists(name[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if verify {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte("Benutzer existiert bereits - anderen Benutzernamen wählen"))
		return
	}

	pw, ok := q["pw"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Password wrong"))
		return
	}
	if len(pw) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Password wrong"))
		return
	}
	if len(pw[0]) < config.LengthPassword {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte(fmt.Sprintln("Neues Passwort zu kurz, Mindestlänge ist", config.LengthPassword)))
		return
	}

	err = database.AddUser(name[0], pw[0], false)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/usermanagement.html#user%s", config.ServerPath, name[0]), http.StatusFound)
}

func usermanagementAdminDeleteUserHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

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
	token, ok := q["token"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}

	name, ok := q["name"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}
	if len(name) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Name wrong"))
		return
	}

	count, err := database.DeleteUser(name[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	c, err := accesstimes.DeleteUser(name[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	count += c

	rw.Write([]byte(fmt.Sprintf("Username: %s\nGelöschte Daten: %d\n%s/usermanagement.html#user%s", name[0], count, config.ServerPrefix, name[0])))
}

func usermanagementAdminDeleteAllInvitationsHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

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
	token, ok := q["token"]
	if !ok {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write([]byte("Invalid token"))
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

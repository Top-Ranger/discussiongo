// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2021,2022 Marcus Soll
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
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
)

type templateInvitationData struct {
	ServerPath   string
	ShowError    bool
	ShowRegister bool
	Error        string
	Invitation   string
	InvitedBy    string
	ForumName    string
	Token        string
	Translation  Translation
}

var (
	invitationTemplate *template.Template
)

func init() {
	var err error

	invitationTemplate, err = template.ParseFS(templateFiles, "template/invitation.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/invitation.html", invitationHandleFunc)
	http.HandleFunc("/registerInvitation.html", registerInvitationHandleFunc)
}

func invitationHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	q := r.URL.Query()
	inv := q.Get("inv")
	if inv == "" {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.InvitationInvalid,
			Invitation:   "",
			InvitedBy:    "",
			ForumName:    config.ForumName,
			Token:        "INVALID",
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	ok, err := database.TestInvitation(inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	td := templateInvitationData{
		ServerPath:   config.ServerPath,
		ShowError:    false,
		ShowRegister: true,
		Error:        "",
		Invitation:   inv,
		InvitedBy:    "",
		ForumName:    config.ForumName,
		Token:        "INVALID",
		Translation:  t,
	}
	if !ok {
		td.Error = "Einladung nicht gültig"
		td.ShowRegister = false
		td.ShowError = true
	}

	invitedby, err := database.GetInvitationCreator(inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	td.InvitedBy = invitedby

	token, err := data.GetStringsTimed(time.Now(), inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	td.Token = token

	err = invitationTemplate.Execute(rw, td)
	if err != nil {
		log.Println("Error executing invitation template:", err)
	}
}

func registerInvitationHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	err := r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.Form

	inv := q.Get("inv")
	if inv == "" {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.InvitationInvalid,
			Invitation:   "",
			InvitedBy:    "",
			ForumName:    config.ForumName,
			Token:        "INVALID",
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	valid, err := database.TestInvitation(inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	if !valid {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.InvitationInvalid,
			Invitation:   "",
			InvitedBy:    "",
			ForumName:    config.ForumName,
			Token:        "INVALID",
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	datenschutzerklärung := q.Get("datenschutzerklärung")
	if datenschutzerklärung != "zugestimmt" {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Invitation:   inv,
			InvitedBy:    "",
			ForumName:    config.ForumName,
			Token:        "INVALID",
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	newToken, err := data.GetStringsTimed(time.Now(), inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	token := q.Get("token")
	if token == "" {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.TokenInvalid,
			Invitation:   inv,
			InvitedBy:    "",
			ForumName:    config.ForumName,
			Token:        newToken,
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	ok := data.VerifyStringsTimed(token, inv, time.Now(), authentificationDuration)
	if !ok {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.TokenInvalid,
			Invitation:   inv,
			InvitedBy:    "",
			ForumName:    config.ForumName,
			Token:        newToken,
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	invitedby, err := database.GetInvitationCreator(inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	name := q.Get("name")
	if len(strings.TrimSpace(name)) == 0 {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.NameInvalid,
			Invitation:   inv,
			InvitedBy:    invitedby,
			ForumName:    config.ForumName,
			Token:        newToken,
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	if protectedUserRegexp.Match([]byte(name)) {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.NameInvalid,
			Invitation:   inv,
			InvitedBy:    invitedby,
			ForumName:    config.ForumName,
			Token:        newToken,
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	valid, err = database.UserExists(name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	if valid {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.UserExists,
			Invitation:   inv,
			InvitedBy:    invitedby,
			ForumName:    config.ForumName,
			Token:        newToken,
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	pw := q.Get("pw")
	if len(pw) < config.LengthPassword {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        fmt.Sprintf(t.PasswortTooShort, config.LengthPassword),
			Invitation:   inv,
			InvitedBy:    invitedby,
			ForumName:    config.ForumName,
			Token:        newToken,
			Translation:  t,
		}
		err := invitationTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing invitation template:", err)
		}
		return
	}

	err = database.RemoveInvitation(inv)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.AddUser(name, pw, false)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.SetInvitedby(name, invitedby, true)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	e := events.Event{
		Type:         EventUserInvited,
		User:         name,
		AffectedUser: invitedby,
		Topic:        eventAdminPseudoTopic,
		Date:         time.Now(),
	}

	_, err = events.SaveEvent(e)
	if err != nil {
		log.Printf("Can not save event %+v: %s", e, err.Error())
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
}

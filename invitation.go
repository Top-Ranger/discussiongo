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
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/database"
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
	inv, ok := q["inv"]
	if !ok {
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
	if len(inv) != 1 {
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

	ok, err := database.TestInvitation(inv[0])
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
		Invitation:   inv[0],
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

	invitedby, err := database.GetInvitationCreator(inv[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	td.InvitedBy = invitedby

	token, err := data.GetStringsTimed(time.Now(), inv[0])
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

	inv, ok := q["inv"]
	if !ok {
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
	if len(inv) != 1 {
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

	valid, err := database.TestInvitation(inv[0])
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

	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	datenschutzerklärung, ok := q["datenschutzerklärung"]
	if !ok {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Invitation:   inv[0],
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
	if len(datenschutzerklärung) != 1 {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Invitation:   inv[0],
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
	if datenschutzerklärung[0] != "zugestimmt" {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: false,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Invitation:   inv[0],
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

	newToken, err := data.GetStringsTimed(time.Now(), inv[0])

	token, ok := q["token"]
	if !ok {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.TokenInvalid,
			Invitation:   inv[0],
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
	if len(token) != 1 {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.TokenInvalid,
			Invitation:   inv[0],
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

	ok = data.VerifyStringsTimed(token[0], inv[0], time.Now(), authentificationDuration)
	if !ok {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.TokenInvalid,
			Invitation:   inv[0],
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

	invitedby, err := database.GetInvitationCreator(inv[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	name, ok := q["name"]
	if !ok {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.NameInvalid,
			Invitation:   inv[0],
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
	if len(name) != 1 {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.NameInvalid,
			Invitation:   inv[0],
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
	if len(strings.TrimSpace(name[0])) == 0 {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.NameInvalid,
			Invitation:   inv[0],
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

	if protectedUserRegexp.Match([]byte(name[0])) {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.NameInvalid,
			Invitation:   inv[0],
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

	valid, err = database.UserExists(name[0])
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
			Invitation:   inv[0],
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

	pw, ok := q["pw"]
	if !ok {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.PasswordInvalid,
			Invitation:   inv[0],
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
	if len(pw) != 1 {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        t.PasswordInvalid,
			Invitation:   inv[0],
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
	if len(pw[0]) < config.LengthPassword {
		td := templateInvitationData{
			ServerPath:   config.ServerPath,
			ShowError:    true,
			ShowRegister: true,
			Error:        fmt.Sprintf(t.PasswortTooShort, config.LengthPassword),
			Invitation:   inv[0],
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

	err = database.RemoveInvitation(inv[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.AddUser(name[0], pw[0], false)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	err = database.SetInvitedby(name[0], invitedby, true)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
}

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
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Top-Ranger/auth/captcha"
	"github.com/Top-Ranger/discussiongo/database"
)

type templateRegisterData struct {
	ServerPath   string
	ForumName    string
	ShowError    bool
	ShowRegister bool
	Error        string
	Captcha      string
	CaptchaID    string
	Translation  Translation
}

var (
	registerTemplate *template.Template
)

func init() {
	b, err := ioutil.ReadFile("template/register.html")
	if err != nil {
		panic(err)
	}
	registerTemplate, err = template.New("register").Parse(string(b))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/register.html", registerHandleFunc)
	http.HandleFunc("/registerUser.html", registerUserHandleFunc)
}

func registerHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	id, captcha, err := captcha.GetStringsTimed(time.Now())
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	captcha = fmt.Sprintf(t.CaptchaString, captcha)

	td := templateRegisterData{
		ServerPath:   config.ServerPath,
		ForumName:    config.ForumName,
		ShowError:    !config.CanRegister,
		ShowRegister: config.CanRegister,
		Error:        t.RegistrationNotPossible,
		Captcha:      captcha,
		CaptchaID:    id,
		Translation:  t,
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = registerTemplate.Execute(rw, td)
	if err != nil {
		log.Println("Error executing registration template:", err)
	}
}

func registerUserHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()

	id, c, err := captcha.GetStringsTimed(time.Now())
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !config.CanRegister {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.RegistrationNotPossible,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	err = r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.Form

	datenschutzerklärung, ok := q["datenschutzerklärung"]
	if !ok {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(datenschutzerklärung) != 1 {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if datenschutzerklärung[0] != "zugestimmt" {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.RegistrationNeedsPrivacyPolicy,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	captchaID, ok := q["captchaID"]
	if !ok {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.CaptchaInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(captchaID) != 1 {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.CaptchaInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	captchaValue, ok := q["captcha"]
	if !ok {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.CaptchaInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(captchaValue) != 1 {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.CaptchaInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	valid := captcha.VerifyStringsTimed(captchaID[0], captchaValue[0], time.Now(), time.Duration(config.CookieMinutes)*time.Minute)
	if !valid {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.CaptchaInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	name, ok := q["name"]
	if !ok {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.NameInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(name) != 1 {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.NameInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(strings.TrimSpace(name[0])) == 0 {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.NameInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	if protectedUserRegexp.Match([]byte(name[0])) {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.NameInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
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
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.UserExists,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	pw, ok := q["pw"]
	if !ok {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.PasswordInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(pw) != 1 {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        t.PasswordInvalid,
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}
	if len(pw[0]) < config.LengthPassword {
		td := templateRegisterData{
			ServerPath:   config.ServerPath,
			ForumName:    config.ForumName,
			ShowError:    true,
			ShowRegister: config.CanRegister,
			Error:        fmt.Sprintf(t.PasswortTooShort, config.LengthPassword),
			Captcha:      c,
			CaptchaID:    id,
			Translation:  t,
		}
		err := registerTemplate.Execute(rw, td)
		if err != nil {
			log.Println("Error executing registration template:", err)
		}
		return
	}

	err = database.AddUser(name[0], pw[0], false)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	log.Println("Registering user", name[0])

	http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
}

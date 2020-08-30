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
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/database"
)

var (
	loginTemplate *template.Template
)

type loginLogoutData struct {
	LoggedIn         bool
	Username         string
	RegisterPossible bool
	ServerPath       string
	ForumName        string
	Token            string
	Translation      Translation
}

func init() {
	b, err := ioutil.ReadFile("template/login.html")
	if err != nil {
		panic(err)
	}
	loginTemplate, err = template.New("login").Parse(string(b))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/login.html", loginPageHandleFunc)
	http.HandleFunc("/login/", loginHandleFunc)
	http.HandleFunc("/logout/", logoutHandleFunc)
}

// SetCookies adds authentification cookies for a given user to the connection represented by a http.ResponseWriter.
// Ater setting those, the user is authenticated and logged in.
// Since the cookies expire after some time (config.CookieMinutes), it is adviced to call it in regular intervals (e.g. whenever the user performs an action).
func SetCookies(rw http.ResponseWriter, username string) error {
	auth, err := data.GetStringsTimed(time.Now(), username)
	if err != nil {
		return err
	}

	cookiePath := config.ServerPath
	if cookiePath == "" {
		cookiePath = "/"
	}

	cookie := http.Cookie{}
	cookie.Name = config.CookieLogin
	cookie.Value = username
	cookie.MaxAge = 60 * config.CookieMinutes
	cookie.Path = cookiePath
	cookie.SameSite = http.SameSiteLaxMode
	cookie.HttpOnly = true
	http.SetCookie(rw, &cookie)

	cookie = http.Cookie{}
	cookie.Name = config.CookieAuth
	cookie.Value = auth
	cookie.MaxAge = 60 * config.CookieMinutes
	cookie.Path = cookiePath
	cookie.SameSite = http.SameSiteLaxMode
	cookie.HttpOnly = true
	http.SetCookie(rw, &cookie)

	return nil
}

// RemoveCookies removes the authentification cookies from a given connection represented by a http.ResponseWriter.
// This has the effect that the user is logged out.
// Please note that the cookies are not invalidated - if they can be restored, the user is logged in again.
func RemoveCookies(rw http.ResponseWriter) {
	cookiePath := config.ServerPath
	if cookiePath == "" {
		cookiePath = "/"
	}

	cookie := http.Cookie{}
	cookie.Name = config.CookieLogin
	cookie.Value = ""
	cookie.Path = cookiePath
	cookie.MaxAge = -60 * config.CookieMinutes
	http.SetCookie(rw, &cookie)

	cookie = http.Cookie{}
	cookie.Name = config.CookieAuth
	cookie.Value = ""
	cookie.Path = cookiePath
	cookie.MaxAge = -60 * config.CookieMinutes
	http.SetCookie(rw, &cookie)
}

// TestUser reports to a given connection represented by *http.Request whether a user is logged in and what his user name is.
func TestUser(r *http.Request) (bool, string) {
	c := r.Cookies()

	u, auth := "", ""

	// username
	for i := range c {
		if c[i].Name == config.CookieLogin {
			u = c[i].Value
		} else if c[i].Name == config.CookieAuth {
			auth = c[i].Value
		}
	}

	if u == "" || auth == "" {
		return false, ""
	}
	return data.VerifyStringsTimed(auth, u, time.Now(), time.Duration(config.CookieMinutes)*time.Minute), u
}

func loginHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	returnError := func() { http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound) }

	err := r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.Form
	token, ok := r.Form["token"]
	if !ok {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token[0], "SYSTEM:UserLogin", time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	user, ok := q["name"]
	if !ok {
		returnError()
		return
	}
	if len(user) != 1 {
		returnError()
		return
	}

	pw, ok := q["pw"]
	if !ok {
		returnError()
		return
	}
	if len(pw) != 1 {
		returnError()
		return
	}

	b, err := database.VerifyUser(user[0], pw[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !b {
		if config.LogFailedLogin {
			log.Printf("Failed login from %s", r.RemoteAddr)
		}
		returnError()
		return
	}

	log.Println("Valid login from", user[0])

	err = database.ModifyLastSeen(user[0])
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	err = SetCookies(rw, user[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
}

func logoutHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	ok, user := TestUser(r)
	if !ok {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err := r.ParseForm()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	token, ok := r.Form["token"]
	if !ok {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	RemoveCookies(rw)
	http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
}

func loginPageHandleFunc(rw http.ResponseWriter, r *http.Request) {
	ok, user := TestUser(r)

	l := loginLogoutData{LoggedIn: ok, Username: user, RegisterPossible: config.CanRegister, ServerPath: config.ServerPath, ForumName: config.ForumName, Translation: GetDefaultTranslation()}

	if l.LoggedIn {
		token, err := data.GetStringsTimed(time.Now(), fmt.Sprintf("%s;Token", user))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		l.Token = token
	} else {
		token, err := data.GetStringsTimed(time.Now(), "SYSTEM:UserLogin")
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		l.Token = token
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	rw.WriteHeader(http.StatusOK)
	err := loginTemplate.Execute(rw, &l)
	if err != nil {
		log.Println("Error executing login template:", err)
	}
}

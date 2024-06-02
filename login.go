// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2022,2024 Marcus Soll
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
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/authtoken"
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
	var err error

	loginTemplate, err = template.ParseFS(templateFiles, "template/login.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/login.html", loginPageHandleFunc)
	http.HandleFunc("/login/", loginHandleFunc)
	http.HandleFunc("/logout/", logoutHandleFunc)
}

// SetCookies adds authentification cookies for a given user to the connection represented by a http.ResponseWriter.
// Ater setting those, the user is authenticated and logged in.
// Token will be refreshed if needed by TestUser().
func SetCookies(rw http.ResponseWriter, username string) error {
	auth, err := authtoken.GetNewToken(username, config.CookieMinutes)
	if err != nil {
		return err
	}

	cookiePath := config.ServerPath
	if cookiePath == "" {
		cookiePath = "/"
	}

	cookie := http.Cookie{}
	cookie.Name = config.CookieLogin
	cookie.Value = auth.ID
	cookie.MaxAge = 60 * config.CookieMinutes
	cookie.Path = cookiePath
	cookie.SameSite = http.SameSiteLaxMode
	cookie.HttpOnly = true
	cookie.Secure = !config.InsecureAllowCookiesOverHTTP
	http.SetCookie(rw, &cookie)

	return nil
}

// RemoveCookies removes the authentification cookies from a given connection represented by a http.ResponseWriter.
// It also removes the associated authtoken from the database.
// This has the effect that the user is logged out.
// Please note that the accesstoken (if present) is invalidated to prevent login if restored.
func RemoveCookies(r *http.Request, rw http.ResponseWriter) error {
	token := ""
	c := r.Cookies()
	for i := range c {
		if c[i].Name == config.CookieLogin {
			token = c[i].Value
		}
	}

	if token != "" {
		err := authtoken.DeleteToken(token)
		if err != nil {
			return err
		}
	}

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

	return nil
}

// TestUser reports to a given connection represented by *http.Request whether a user is logged in and what his user name is.
// Will refresh cookie and authtoken when needed.
func TestUser(r *http.Request, rw http.ResponseWriter) (bool, string) {
	c := r.Cookies()

	auth := ""

	// username
	for i := range c {
		if c[i].Name == config.CookieLogin {
			auth = c[i].Value
		}
	}

	if auth == "" {
		return false, ""
	}
	user, validUntil, ok := authtoken.CheckUser(auth)

	if !ok {
		return false, ""
	}
	if time.Until(validUntil)/time.Minute < time.Duration(config.CookieMinutes)/2 {
		// Refresh token
		log.Println("refresh")
		err := SetCookies(rw, user)
		if err != nil {
			log.Println("login: can not refresh cookie:", err)
		}
	}
	return true, user
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
	token := q.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token, "SYSTEM:UserLogin", time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	user := q.Get("name")
	if user == "" {
		returnError()
		return
	}

	pw := q.Get("pw")
	if pw == "" {
		returnError()
		return
	}

	b, err := database.VerifyUser(user, pw)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	if !b {
		if config.LogFailedLogin {
			log.Printf("Failed login from %s", GetRealIP(r))
		}
		returnError()
		return
	}

	log.Println("Valid login from", user)

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	err = SetCookies(rw, user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
}

func logoutHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	ok, user := TestUser(r, rw)
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

	token := r.Form.Get("token")
	if token == "" {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token, fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	RemoveCookies(r, rw)
	http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
}

func loginPageHandleFunc(rw http.ResponseWriter, r *http.Request) {
	ok, user := TestUser(r, rw)

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

// GetRealIP tries to fing the real IP address of a client.
// If an error is found, that error will be returned instead of an IP address.
// A reverse proxy is only assumed if address is a loopback device (to avoid spoofing)
func GetRealIP(r *http.Request) string {
	ipPart, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return err.Error()
	}
	ip := net.ParseIP(strings.SplitN(ipPart, "%", 1)[0])
	if ip != nil && !ip.IsLoopback() {
		// We found a valid IP, this is most likely correct
		goto returnIP
	}

	// This is likely behind a reverse proxy, try to find real address
	{
		header := r.Header.Get("Forwarded")
		if header != "" {
			headerParts := strings.Split(header, ";")
			for i := range headerParts {
				// Find for part of header
				if strings.HasPrefix(headerParts[i], "for=") {
					headerParts[i] = strings.TrimPrefix(headerParts[i], "for=")
					ip = processSplittedHeader(strings.Split(headerParts[i], ","))
					if ip != nil {
						goto returnIP
					}
				}
			}
		}
	}
	{
		header := r.Header.Get("X-Forwarded-For")
		if header != "" {
			ip = processSplittedHeader(strings.Split(header, ","))
			if ip != nil {
				goto returnIP
			}

		}
	}

returnIP:
	if ip == nil {
		return "unknown IP"
	}
	return ip.String()
}

func processSplittedHeader(split []string) net.IP {
	for i := len(split) - 1; i >= 0; i++ {
		// Go back to forward to find irst non local address. This way, fake addresses can't be spoofed by sending header (assumed proxy is trusted)

		// Assume entry is an IP address. Handle other cases later.
		s := split[i]
		// Case IPv6 with brackets - should do nothing in all other cases
		s = strings.TrimPrefix(s, "[")
		s = strings.TrimSuffix(s, "]")
		ip := net.ParseIP(strings.SplitN(s, "%", 1)[0])

		if ip == nil {
			// Maybe has form host:port?
			ipPart, _, err := net.SplitHostPort(split[i])
			if err != nil {
				// Invalid entry - something is wrong, stop processing
				return nil
			}
			ip := net.ParseIP(strings.SplitN(ipPart, "%", 1)[0])
			if ip == nil {
				// Invalid entry - something is wrong, stop processing
				return nil
			}
		}
		if ip != nil && !ip.IsLoopback() {
			return ip
		}
	}
	return nil
}

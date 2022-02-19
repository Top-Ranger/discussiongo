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
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
	"github.com/Top-Ranger/discussiongo/files"
)

type impressumConfigStruct struct {
	ImpressumPath string
	DSGVOPath     string
}

type impressumStruct struct {
	Text        template.HTML
	ServerPath  string
	ForumName   string
	Translation Translation
}

// DSGVOExport represents all information needed for an export according toDSGVO Art. 15 / DSGVO Art. 20.
// It can then be marshalled e.g. to XMLS.
type DSGVOExport struct {
	XMLName        xml.Name `xml:"export"`
	User           database.User
	Topics         []database.Topic
	Posts          []database.Post
	Files          []files.File
	Events         []events.Event
	InvitedUser    []DSGVOExportInvitedUsers
	Invitations    []string
	TopicsLastRead []accesstimes.AccessTimes
	NotExported    []string
}

// DSGVOExportInvitedUsers is a helper struct for DSGVOExport, which represents an invided user.
type DSGVOExportInvitedUsers struct {
	Username string
	Direct   bool
}

var (
	impressumConfig     = impressumConfigStruct{}
	impressum           = impressumStruct{}
	impressumTemplate   *template.Template
	dsgvo               = impressumStruct{}
	dsgvoTemplate       *template.Template
	completeDSGVOStruct = sync.Once{}
)

func init() {
	ic, err := loadImpressum("./impressum.json")
	if err != nil {
		panic(err)
	}

	impressumConfig = ic

	impressumTemplate, err = template.ParseFS(templateFiles, "template/impressum.html")
	if err != nil {
		panic(err)
	}

	dsgvoTemplate, err = template.ParseFS(templateFiles, "template/dsgvo.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/impressum.html", impressumHandleFunc)
	http.HandleFunc("/datenschutz.html", dsgvoHandleFunc)
	http.HandleFunc("/dsgvoExport.xml", dsgvoExportHandleFunc)
}

func funcCompleteDSGVOStruct() {
	t := GetDefaultTranslation()

	b, err := os.ReadFile(impressumConfig.ImpressumPath)
	if err != nil {
		panic(err)
	}
	impressum = impressumStruct{
		Text:        formatPost(string(b)),
		ServerPath:  config.ServerPath,
		ForumName:   config.ForumName,
		Translation: t,
	}

	b, err = os.ReadFile(impressumConfig.DSGVOPath)
	if err != nil {
		panic(err)
	}
	dsgvo = impressumStruct{
		Text:        formatPost(string(b)),
		ServerPath:  config.ServerPath,
		ForumName:   config.ForumName,
		Translation: t,
	}
}

func loadImpressum(path string) (impressumConfigStruct, error) {
	log.Println("Loading impressum")
	b, err := os.ReadFile(path)
	if err != nil {
		return impressumConfigStruct{}, errors.New(fmt.Sprintln("Can not read config.json:", err))
	}

	i := impressumConfigStruct{}
	err = json.Unmarshal(b, &i)
	if err != nil {
		return impressumConfigStruct{}, errors.New(fmt.Sprintln("Error while parsing config.json:", err))
	}

	return i, nil
}

func impressumHandleFunc(rw http.ResponseWriter, r *http.Request) {
	completeDSGVOStruct.Do(funcCompleteDSGVOStruct)

	rw.WriteHeader(http.StatusOK)
	err := impressumTemplate.Execute(rw, impressum)
	if err != nil {
		log.Println("Error executing impressum template:", err)
	}
}

func dsgvoHandleFunc(rw http.ResponseWriter, r *http.Request) {
	completeDSGVOStruct.Do(funcCompleteDSGVOStruct)

	rw.WriteHeader(http.StatusOK)
	err := dsgvoTemplate.Execute(rw, dsgvo)
	if err != nil {
		log.Println("Error executing dsgvo template:", err)
	}
}

func dsgvoExportHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r)

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

	dsgvo := DSGVOExport{}
	var err error

	dsgvo.User, err = database.GetUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	dsgvo.Topics, err = database.GetTopicsByUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	dsgvo.Posts, err = database.GetPostsByUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	dsgvo.Files, err = files.GetFilesForUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	dsgvo.Events, err = events.GetEventsOfUser(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	users, err := database.GetAllUser()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	dsgvo.InvitedUser = make([]DSGVOExportInvitedUsers, 0, len(users))
	for i := range users {
		if users[i].InvidedBy == user {
			dsgvo.InvitedUser = append(dsgvo.InvitedUser, DSGVOExportInvitedUsers{Username: users[i].Name, Direct: users[i].InvitationDirect})
		}
	}

	dsgvo.Invitations, err = database.GetInvitations(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	dsgvo.TopicsLastRead, err = accesstimes.GetUserTimes(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	dsgvo.NotExported = []string{"hashed password", "salt for password hash"}

	b, err := xml.MarshalIndent(&dsgvo, "", "\t")
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	rw.Write([]byte(xml.Header))
	rw.Write(b)
}

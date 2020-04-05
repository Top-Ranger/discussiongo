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
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
)

type impressumStruct struct {
	DSGVOResponisble   string
	Name               string
	Address            string
	Mail               string
	SecondContactType  string
	SecondContactValue string
	Additions          string
	ShowAdditions      bool
	ServerPath         string
	ForumName          string
}

// DSGVOExport represents all information needed for an export according toDSGVO Art. 15 / DSGVO Art. 20.
// It can then be marshalled e.g. to XMLS.
type DSGVOExport struct {
	XMLName        xml.Name `xml:"export"`
	User           database.User
	Topics         []database.Topic
	Posts          []database.Post
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
	impressum           = impressumStruct{}
	impressumTemplate   *template.Template
	dsgvoTemplate       *template.Template
	completeDSGVOStruct = sync.Once{}
)

func init() {
	i, err := loadImpressum("./impressum.json")
	if err != nil {
		panic(err)
	}

	impressum = i

	b, err := ioutil.ReadFile("template/impressum.html")
	if err != nil {
		panic(err)
	}
	impressumTemplate, err = template.New("impressum").Parse(string(b))
	if err != nil {
		panic(err)
	}

	b, err = ioutil.ReadFile("template/dsgvo.html")
	if err != nil {
		panic(err)
	}
	dsgvoTemplate, err = template.New("dsgvo").Parse(string(b))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/impressum.html", impressumHandleFunc)
	http.HandleFunc("/datenschutz.html", dsgvoHandleFunc)
	http.HandleFunc("/dsgvoExport.xml", dsgvoExportHandleFunc)
}

func funcCompleteDSGVOStruct() {
	impressum.ServerPath = config.ServerPath
	impressum.ForumName = config.ForumName
}

func loadImpressum(path string) (impressumStruct, error) {
	log.Println("Loading impressum")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return impressumStruct{}, errors.New(fmt.Sprintln("Can not read config.json:", err))
	}

	i := impressumStruct{}
	err = json.Unmarshal(b, &i)
	if err != nil {
		return impressumStruct{}, errors.New(fmt.Sprintln("Error while parsing config.json:", err))
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
	err := dsgvoTemplate.Execute(rw, impressum)
	if err != nil {
		log.Println("Error executing dsgvo template:", err)
	}
}

func dsgvoExportHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
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

	rw.Write(b)
}

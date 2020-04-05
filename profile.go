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

	"github.com/Top-Ranger/discussiongo/database"
)

type templateProfileData struct {
	ServerPath string
	ForumName  string
	User       string
	Comment    template.HTML
	HasComment bool
	Topics     []topicData
	Posts      []postData
}

var (
	profileTemplate *template.Template
)

func init() {
	funcMap := template.FuncMap{
		"even": func(i int) bool {
			return i%2 == 0
		},
	}

	b, err := ioutil.ReadFile("template/profile.html")
	if err != nil {
		panic(err)
	}
	profileTemplate, err = template.New("profile").Funcs(funcMap).Parse(string(b))
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/profile.html", profileHandleFunc)
}

func profileHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, _ := TestUser(r)

	if !config.CanReadWithoutRegister && !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
	}

	q := r.URL.Query()
	quser, ok := q["user"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(quser) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	u, err := database.GetUser(quser[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	topics, err := database.GetTopicsByUser(u.Name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	posts, err := database.GetPostsByUser(u.Name)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	td := templateProfileData{
		ServerPath: config.ServerPath,
		ForumName:  config.ForumName,
		User:       u.Name,
		Comment:    formatPost(u.Comment),
		HasComment: u.Comment != "",
		Topics:     make([]topicData, 0, len(topics)),
		Posts:      make([]postData, 0, len(posts)),
	}

	for i := range topics {
		td.Topics = append(td.Topics, topicData{
			ID:       topics[i].ID,
			Name:     topics[i].Name,
			Modified: topics[i].LastModified.Format(time.RFC822),
			Creator:  topics[i].Creator,
			Closed:   topics[i].Closed,
			Pinned:   topics[i].Pinned,
			New:      false, // Not used here
		})
	}

	for i := range posts {
		t, err := database.GetTopic(posts[i].TopicID)
		if err != nil {
			print(posts[i].TopicID)
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		td.Posts = append(td.Posts, postData{
			ID:         posts[i].ID,
			TID:        posts[i].TopicID,
			TName:      t.Name,
			Content:    formatPost(posts[i].Content),
			RawContent: posts[i].Content,
			Date:       posts[i].Time.Format(time.RFC822),
			Creator:    posts[i].Poster,
			New:        false, // not used here
			CanDelete:  false, // not used here
		})
	}

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = profileTemplate.Execute(rw, td)
	if err != nil {
		log.Println("Error executing post template:", err)
	}
}

// SPDX-License-Identifier: Apache-2.0
// Copyright 2020,2021 Marcus Soll
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
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Top-Ranger/auth/data"
	"github.com/Top-Ranger/discussiongo/accesstimes"
	"github.com/Top-Ranger/discussiongo/database"
	"github.com/Top-Ranger/discussiongo/events"
	"github.com/Top-Ranger/discussiongo/files"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/microcosm-cc/bluemonday"
)

type templatePostData struct {
	ServerPath        string
	ServerPrefix      string
	ForumName         string
	LoggedIn          bool
	User              string
	IsAdmin           bool
	Topic             string
	TopicID           string
	Closed            bool
	CanClose          bool
	Pinned            bool
	CanRename         bool
	HasNew            bool
	CanSaveFiles      bool
	CurrentUpdate     int64
	Timeline          []timelineData
	Token             string
	FileUploadMessage string
	Translation       Translation
}

type timelineData struct {
	Time  time.Time
	Post  *postData
	File  *fileData
	Event *eventData
}

type postData struct {
	ID         string
	TID        string
	TName      string
	Content    template.HTML
	RawContent string
	Date       string
	Creator    string
	New        bool
	CanDelete  bool
}

type fileData struct {
	ID        string
	Name      string
	User      string
	Topic     string
	Date      string
	CanDelete bool
	New       bool
}

var (
	postTemplate *template.Template
	policy       *bluemonday.Policy
)

func init() {
	var err error

	policy = bluemonday.NewPolicy()
	policy.AllowElements("a", "b", "blockquote", "br", "caption", "code", "del", "em", "h1", "h2", "h3", "h4", "h5", "h6", "hr", "i", "ins", "kbd", "mark", "p", "pre", "q", "s", "samp", "strong", "sub", "sup", "u")
	policy.AllowAttrs("disabled", "type", "checked").OnElements("input")
	policy.AllowLists()
	policy.AllowStandardURLs()
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("class").OnElements("code")
	policy.RequireNoReferrerOnLinks(true)
	policy.AllowTables()
	policy.AddTargetBlankToFullyQualifiedLinks(true)
	policy.RequireNoFollowOnFullyQualifiedLinks(true)

	postTemplate, err = template.New("posts").Funcs(evenOddFuncMap).ParseFS(templateFiles, "template/posts.html")
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/topic.html", postHandleFunc)
	http.HandleFunc("/newPost.html", newPostHandleFunc)
	http.HandleFunc("/deletePost.html", deletePostHandleFunc)
	http.HandleFunc("/getFormattedPost/", getFormattedPostHandleFunc)
}

func formatPost(s string) template.HTML {
	var buf bytes.Buffer
	md := goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(html.WithHardWraps()))
	err := md.Convert([]byte(s), &buf)
	if err != nil {
		return template.HTML(policy.Sanitize(fmt.Sprintf("Error rendering markdown: %s", err.Error())))
	}
	return template.HTML(policy.SanitizeBytes(buf.Bytes()))
}

func postHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !config.CanReadWithoutRegister && !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
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
		SetCookies(rw, user)
	}

	q := r.URL.Query()
	id, ok := q["id"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(id) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic, err := database.GetTopic(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	posts, err := database.GetPosts(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	fs, err := files.GetFileMetadataOfTopic(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	events, err := events.GetEventsOfTopic(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	td := templatePostData{
		ServerPath:        config.ServerPath,
		ServerPrefix:      config.ServerPrefix,
		ForumName:         config.ForumName,
		LoggedIn:          loggedIn,
		User:              user,
		IsAdmin:           isAdmin,
		Topic:             topic.Name,
		TopicID:           id[0],
		Closed:            topic.Closed,
		CanClose:          (isAdmin || user == topic.Creator),
		Pinned:            topic.Pinned,
		CanRename:         (isAdmin || user == topic.Creator),
		HasNew:            false,
		CanSaveFiles:      config.EnableFileUpload || (isAdmin && config.EnableFileUploadAdmin),
		CurrentUpdate:     database.GetLastUpdateTopicPost(),
		Timeline:          make([]timelineData, 0, len(posts)+len(fs)+len(events)),
		FileUploadMessage: config.FileUploadMessage,
		Translation:       GetDefaultTranslation(),
	}

	var lastUpdate time.Time

	if loggedIn {
		token, err := data.GetStringsTimed(time.Now(), fmt.Sprintf("%s;Token", user))
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		td.Token = token

		u, err := database.GetUser(user)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		l, err := accesstimes.GetTimes(u.Name, []string{topic.ID})
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
		lastUpdate = l[0]

		err = accesstimes.SaveTime(user, topic.ID, time.Now())
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}
	}

	for i := range posts {
		p := postData{
			ID:         posts[i].ID,
			TID:        posts[i].TopicID,
			TName:      "unimportant",
			Content:    formatPost(posts[i].Content),
			RawContent: posts[i].Content,
			Date:       posts[i].Time.Format(time.RFC822),
			Creator:    posts[i].Poster,
			New:        false,
			CanDelete:  (isAdmin || user == posts[i].Poster),
		}
		if loggedIn {
			if lastUpdate.Before(posts[i].Time) {
				p.New = true
				td.HasNew = true
			}
		}
		td.Timeline = append(td.Timeline, timelineData{
			Time: posts[i].Time,
			Post: &p,
		})
	}

	for i := range fs {
		f := fileData{
			ID:        fs[i].ID,
			Name:      fs[i].Name,
			User:      fs[i].User,
			Date:      fs[i].Date.Format(time.RFC822),
			CanDelete: (isAdmin || user == fs[i].User),
			New:       false,
		}
		if loggedIn {
			if lastUpdate.Before(fs[i].Date) {
				f.New = true
				td.HasNew = true
			}
		}
		td.Timeline = append(td.Timeline, timelineData{
			Time: fs[i].Date,
			File: &f,
		})
	}

	for i := range events {
		e := eventToEventData(events[i])
		if loggedIn {
			if lastUpdate.Before(events[i].Date) {
				e.New = true
				td.HasNew = true
			}
		}
		td.Timeline = append(td.Timeline, timelineData{
			Time:  events[i].Date,
			Event: &e,
		})
	}

	sort.Slice(td.Timeline, func(i, j int) bool { return td.Timeline[i].Time.Before(td.Timeline[j].Time) })

	rw.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	err = postTemplate.ExecuteTemplate(rw, "posts.html", td)
	if err != nil {
		log.Println("Error executing post template:", err)
	}
}

func newPostHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r)

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
	post, ok := q["post"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(post) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(strings.TrimSpace(post[0])) == 0 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	id, ok := q["tid"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(id) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	topic, err := database.GetTopic(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	if topic.Closed {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TopicIsClosed))
		return
	}

	token, ok := q["token"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(token) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	postID, err := database.AddPost(id[0], user, post[0])
	if err != nil {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	err = database.TopicModifyTime(id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/topic.html?id=%s#post%s", config.ServerPath, id[0], postID), http.StatusFound)
}

func deletePostHandleFunc(rw http.ResponseWriter, r *http.Request) {
	loggedIn, user := TestUser(r)

	if !loggedIn {
		http.Redirect(rw, r, fmt.Sprintf("%s/login.html", config.ServerPath), http.StatusFound)
		return
	}

	isAdmin, err := database.IsAdmin(user)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.URL.Query()
	id, ok := q["id"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(id) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	tid, ok := q["tid"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(tid) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	token, ok := q["token"]
	if !ok {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	if len(token) != 1 {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	post, err := database.GetSinglePost(id[0])
	if err != nil {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	if !isAdmin && user != post.Poster {
		http.Redirect(rw, r, fmt.Sprintf("%s/", config.ServerPath), http.StatusFound)
		return
	}

	err = database.DeletePost(tid[0], id[0])
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}

	_, err = events.SaveEvent(events.Event{
		Type:  EventPostDeleted,
		User:  user,
		Topic: post.TopicID,
		Date:  post.Time,
	})

	err = database.ModifyLastSeen(user)
	if err != nil {
		log.Println("Can not modify last seen:", err)
	}

	http.Redirect(rw, r, fmt.Sprintf("%s/topic.html?id=%s", config.ServerPath, tid[0]), http.StatusFound)
}

func getFormattedPostHandleFunc(rw http.ResponseWriter, r *http.Request) {
	t := GetDefaultTranslation()
	loggedIn, user := TestUser(r)

	if !loggedIn {
		rw.WriteHeader(http.StatusForbidden)
		return
	}

	err := r.ParseMultipartForm(10000000) // 10 MB
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(err.Error()))
		return
	}

	q := r.Form

	token, ok := q["token"]
	if !ok {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	if len(token) != 1 {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte(t.TokenInvalid))
		return
	}
	valid := data.VerifyStringsTimed(token[0], fmt.Sprintf("%s;Token", user), time.Now(), authentificationDuration)
	if !valid {
		rw.WriteHeader(http.StatusForbidden)
		rw.Write([]byte(t.TokenInvalid))
		return
	}

	post, ok := q["post"]
	if !ok {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	if len(post) != 1 {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	rw.Write([]byte(formatPost(post[0])))
}

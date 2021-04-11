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
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed static font js css
var cachedFiles embed.FS
var cssTemplates *template.Template

var (
	etag              = fmt.Sprint("\"", strconv.FormatInt(time.Now().Unix(), 10), "\"")
	etagCompareApache = ""
	etagCompareCaddy  = ""
)

func init() {
	var err error

	cssTemplates, err = template.ParseFS(cachedFiles, "css/*")
	if err != nil {
		panic(err)
	}

	etagCompare := strings.TrimSuffix(etag, "\"")
	etagCompareApache = strings.Join([]string{etagCompare, "-"}, "")       // Dirty hack for apache2, who appends -gzip inside the quotes if the file is compressed, thus preventing If-None-Match matching the ETag
	etagCompareCaddy = strings.Join([]string{"W/", etagCompare, "\""}, "") // Dirty hack for caddy, who appends W/ before the quotes if the file is compressed, thus preventing If-None-Match matching the ETag

	http.HandleFunc("/static/", fileHandleFunc)
	http.HandleFunc("/font/", fileHandleFunc)
	http.HandleFunc("/js/", fileHandleFunc)
	http.HandleFunc("/css/", fileHandleFunc)
}

func fileHandleFunc(rw http.ResponseWriter, r *http.Request) {
	// Check for ETag
	v, ok := r.Header["If-None-Match"]
	if ok {
		for i := range v {
			if v[i] == etag || v[i] == etagCompareCaddy || strings.HasPrefix(v[i], etagCompareApache) {
				rw.WriteHeader(http.StatusNotModified)
				return
			}
		}
	}

	// Send file if existing in cache
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/")

	if strings.HasPrefix(path, "css/") {
		// special case
		path = strings.TrimPrefix(path, "css/")
		rw.Header().Set("ETag", etag)
		rw.Header().Set("Cache-Control", "public, max-age=43200")
		rw.Header().Set("Content-Type", "text/css")
		err := cssTemplates.ExecuteTemplate(rw, path, struct{ ServerPath string }{config.ServerPath})
		if err != nil {
			log.Println("server:", err)
		}
		return
	}

	data, err := cachedFiles.Open(path)
	if err != nil {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		rw.Header().Set("ETag", etag)
		rw.Header().Set("Cache-Control", "public, max-age=43200")
		switch {
		case strings.HasSuffix(path, ".svg"):
			rw.Header().Set("Content-Type", "image/svg+xml")
		case strings.HasSuffix(path, ".css"):
			rw.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(path, ".ttf"):
			rw.Header().Set("Content-Type", "application/x-font-truetype")
		case strings.HasSuffix(path, ".js"):
			rw.Header().Set("Content-Type", "application/javascript")
		default:
			rw.Header().Set("Content-Type", "text/plain")
		}
		io.Copy(rw, data)
	}
}

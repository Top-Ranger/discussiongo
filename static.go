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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CSSData is a helper struct to include the server path into CSS files.
type CSSData struct {
	ServerPath string
}

var (
	// Stores files from css/ static/ and font/ in memory
	cachedFiles = make(map[string][]byte, 50)
	cachedCSS   = make(map[string]*template.Template, 50)

	etag              = fmt.Sprint("\"", strconv.FormatInt(time.Now().Unix(), 10), "\"")
	etagCompareApache = ""
	etagCompareCaddy  = ""
)

func init() {
	etag := fmt.Sprint("\"", strconv.FormatInt(time.Now().Unix(), 10), "\"")
	etagCompare := strings.TrimSuffix(etag, "\"")
	etagCompareApache = strings.Join([]string{etagCompare, "-"}, "")       // Dirty hack for apache2, who appends -gzip inside the quotes if the file is compressed, thus preventing If-None-Match matching the ETag
	etagCompareCaddy = strings.Join([]string{"W/", etagCompare, "\""}, "") // Dirty hack for caddy, who appends W/ before the quotes if the file is compressed, thus preventing If-None-Match matching the ETag

	for _, d := range []string{"static/", "font/", "js/"} {
		filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Panicln("Error wile caching files:", err)
			}

			if info.Mode().IsRegular() {
				log.Println("Caching file", path)

				b, err := ioutil.ReadFile(path)
				if err != nil {
					log.Println("Error reading file:", err)
					return err
				}
				cachedFiles[path] = b
				return nil
			}
			log.Println("Not caching", path)
			return nil
		})
	}

	for _, d := range []string{"css/"} {
		filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Panicln("Error wile caching files:", err)
			}

			if info.Mode().IsRegular() {
				log.Println("Caching file", path)

				b, err := ioutil.ReadFile(path)
				if err != nil {
					log.Println("Error reading file:", err)
					return err
				}
				t, err := template.New(path).Parse(string(b))
				if err != nil {
					log.Println("Error reading file (parsing):", err)
					return err
				}
				cachedCSS[path] = t

				return nil
			}
			log.Println("Not caching", path)
			return nil
		})
	}

	http.HandleFunc("/static/", fileHandleFunc)
	http.HandleFunc("/font/", fileHandleFunc)
	http.HandleFunc("/js/", fileHandleFunc)
	http.HandleFunc("/css/", cssHandleFunc)
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
	data, ok := cachedFiles[path]
	if !ok {
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
		default:
			rw.Header().Set("Content-Type", "text/plain")
		}
		rw.Write(data)
	}
}

func cssHandleFunc(rw http.ResponseWriter, r *http.Request) {
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
	data, ok := cachedCSS[path]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
	} else {
		rw.Header().Set("ETag", etag)
		rw.Header().Set("Cache-Control", "public, max-age=43200")
		rw.Header().Set("Content-Type", "text/css")

		c := CSSData{
			ServerPath: config.ServerPath,
		}

		err := data.Execute(rw, c)
		if err != nil {
			log.Println("Error executing CSS template:", err)
		}
	}
}

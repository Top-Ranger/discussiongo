// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Marcus Soll
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
	"html/template"
	"time"

	"github.com/Top-Ranger/discussiongo/events"
)

const (
	EventCloseTopic = iota
	EventOpenTopic
	EventPinTopic
	EventUnpinTopic
)

type eventData struct {
	ID          string
	Description template.HTML
	User        string
	Topic       string
	Date        string
	New         bool
	RealUser    bool
}

func eventToEventData(e events.Event) eventData {
	// No new is set
	tl := GetDefaultTranslation()
	ed := eventData{
		ID:       e.ID,
		User:     e.User,
		Topic:    e.Topic,
		Date:     e.Date.Format(time.RFC822),
		RealUser: e.User != events.AnoymousUser,
	}
	switch e.Type {
	case EventCloseTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventCloseTopic))
	case EventOpenTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventOpenTopic))
	case EventPinTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventPinTopic))
	case EventUnpinTopic:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.EventUnpitTopic))
	default:
		ed.Description = template.HTML(template.HTMLEscapeString(tl.UnknownEvent))
	}
	return ed
}

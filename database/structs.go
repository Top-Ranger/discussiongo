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

package database

import "time"

// User represents a user in the database.
// For security reasons, the password and the salt is not included.
type User struct {
	Name             string
	Admin            bool
	Comment          string
	InvidedBy        string
	InvitationDirect bool
	LastSeen         time.Time
}

// Topic represents a topic in the database.
type Topic struct {
	ID           string
	Name         string
	Creator      string
	Created      time.Time
	LastModified time.Time
	Closed       bool
	Pinned       bool
}

// Post represents a post in the database.
type Post struct {
	ID      string
	TopicID string
	Poster  string
	Content string
	Time    time.Time
}

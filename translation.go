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
	"embed"
	"encoding/json"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

// Translation represents an object holding all translations
type Translation struct {
	Language                       string
	CreatedBy                      string
	Impressum                      string
	PrivacyPolicy                  string
	Back                           string
	InvitationMessage              string
	InvitedBy                      string
	Name                           string
	Password                       string
	IAcceptPrivacyPolicy           string
	RegisterNow                    string
	Login                          string
	Logout                         string
	NewPostTopicReloadMessage      string
	NewPostTopicMessage            string
	NavigateToBottom               string
	Topic                          string
	Topics                         string
	Posts                          string
	Post                           string
	Closed                         string
	Pinned                         string
	New                            string
	Creator                        string
	CreatedAt                      string
	CopyLink                       string
	CopyContent                    string
	DeletePost                     string
	CloseTopic                     string
	OpenTopic                      string
	UnpinTopic                     string
	PinTopic                       string
	User                           string
	NewPost                        string
	Preview                        string
	CreatePost                     string
	Profile                        string
	Comment                        string
	GoToPost                       string
	CaptchaString                  string
	ActivateNotifications          string
	NotificationsActivated         string
	JavaScriptWarning              string
	UserSettings                   string
	UserManagement                 string
	MarkAllRead                    string
	NewTopic                       string
	CreateTopic                    string
	PinnedTopics                   string
	ClosedTopics                   string
	LastChange                     string
	DeleteTopic                    string
	UserIsAdministrator            string
	LastActicity                   string
	CurrentComment                 string
	ChangeComment                  string
	ChangePassword                 string
	OldPassword                    string
	NewPassword                    string
	Invitations                    string
	OpenInvitations                string
	DeleteInvitation               string
	NewInvitation                  string
	ExportDataShort                string
	ExportDataLong                 string
	DeleteUser                     string
	DeleteUserWarning              string
	UserList                       string
	Indirect                       string
	SetAdministrator               string
	RemoveAdministrator            string
	ResetPassword                  string
	RegisterUser                   string
	DeleteAllInvitation            string
	InvitationInvalid              string
	RegistrationNeedsPrivacyPolicy string
	TokenInvalid                   string
	NameInvalid                    string
	UserExists                     string
	PasswordInvalid                string
	PasswortTooShort               string
	RegistrationNotPossible        string
	CaptchaInvalid                 string
	TopicIsClosed                  string
	OldPasswordWrong               string
	InvalidRequest                 string
	Deleted                        string
	LoginRequired                  string
	ErrorOccured                   string
	NewFile                        string
	DeleteFile                     string
	UploadFile                     string
	Files                          string
	FileTooLarge                   string
	Size                           string
	RenameTopic                    string
	Event                          string
	DeleteEvent                    string
	UnknownEvent                   string
	AdminEvents                    string
	EventCloseTopic                string
	EventOpenTopic                 string
	EventPinTopic                  string
	EventUnpitTopic                string
	EventTopicRenamed              string
	EventPostDeleted               string
	EventFileDeleted               string
	EventUserRegistered            string
	EventUserInvited               string
	EventUserDeleted               string
	EventUserAdminDeleted          string
	EventTopicDeleted              string
	EventUserRegisteredByAdmin     string
	EventSetAdministrator          string
	EventRemoveAdministrator       string
}

const defaultLanguage = "de"

//go:embed translation
var translationFiles embed.FS

var initialiseCurrent sync.Once
var current Translation
var rwlock sync.RWMutex
var translationPath = "./translation"

// GetTranslation returns a Translation struct of the given language.
// This function always loads translations from disk. Try to use GetDefaultTranslation where possible.
func GetTranslation(language string) (Translation, error) {
	t, err := getSingleTranslation(language)
	if err != nil {
		return Translation{}, err
	}
	d, err := getSingleTranslation(defaultLanguage)
	if err != nil {
		return Translation{}, err
	}

	// Set unknown strings to default value
	vp := reflect.ValueOf(&t)
	dv := reflect.ValueOf(d)
	v := vp.Elem()

	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).CanSet() {
			continue
		}
		if v.Field(i).Kind() != reflect.String {
			continue
		}
		if v.Field(i).String() == "" {
			v.Field(i).SetString(dv.Field(i).String())
		}
	}
	return t, nil
}

func getSingleTranslation(language string) (Translation, error) {
	if language == "" {
		return GetDefaultTranslation(), nil
	}

	file := strings.Join([]string{language, "json"}, ".")
	file = filepath.Join(translationPath, file)

	b, err := translationFiles.ReadFile(file)
	if err != nil {
		return Translation{}, err
	}
	t := Translation{}
	err = json.Unmarshal(b, &t)
	if err != nil {
		return Translation{}, err
	}
	return t, nil
}

// SetDefaultTranslation sets the default language to the provided one.
// Does nothing if it returns error != nil.
func SetDefaultTranslation(language string) error {
	if language == "" {
		return nil
	}

	t, err := GetTranslation(language)
	rwlock.Lock()
	defer rwlock.Unlock()
	if err != nil {
		return err
	}
	current = t
	return nil
}

// GetDefaultTranslation returns a Translation struct of the current default language.
func GetDefaultTranslation() Translation {
	initialiseCurrent.Do(func() {
		rwlock.RLock()
		c := current.Language
		rwlock.RUnlock()
		if c == "" {
			err := SetDefaultTranslation(defaultLanguage)
			if err != nil {
				log.Printf("Can not load default language (%s): %s", defaultLanguage, err.Error())
			}
		}
	})
	rwlock.RLock()
	defer rwlock.RUnlock()
	return current
}

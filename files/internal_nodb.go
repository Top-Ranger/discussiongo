//go:build !sqlite && !mysql

// SPDX-License-Identifier: Apache-2.0
// Copyright 2021,2022 Marcus Soll
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

package files

import "errors"

// InitDB initialises the database.
// Must be called before any other function.
// This stub will return an error if no build tags are set.
func InitDB(config string) error {
	return errors.New("files: no database type selected at compile time")
}

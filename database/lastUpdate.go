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

import (
	"sync"
	"time"
)

var (
	lastUpdateTopicPost      int64 = time.Now().UnixNano()
	lastUpdateTopicPostMutex       = sync.RWMutex{}
)

func setLastUpdateTopicPost() {
	lastUpdateTopicPostMutex.Lock()
	defer lastUpdateTopicPostMutex.Unlock()
	lastUpdateTopicPost = time.Now().UnixNano()
}

// GetLastUpdateTopicPost returns the time of the last update of any post/topic as an int64 in nano seconds.
func GetLastUpdateTopicPost() int64 {
	lastUpdateTopicPostMutex.RLock()
	defer lastUpdateTopicPostMutex.RUnlock()
	return lastUpdateTopicPost
}

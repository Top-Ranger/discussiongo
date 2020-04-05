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

package accesstimes

import (
	"log"
	"sync"
	"time"
)

type save struct {
	Name  string
	Topic int
	Time  time.Time
}

var saveTime = make(chan save, 10)
var deleteUser = make(chan string)
var waitWrite = sync.Cond{}

func init() {
	waitWrite.L = new(sync.Mutex)
}

func worker() {
	buffer := make([]*save, 0, 100)
	t := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-t.C:
			if len(buffer) != 0 {
				tx, err := db.Begin()
				if err != nil {
					log.Println("Can not begin transaction:", err)
				}
				for i := range buffer {
					if buffer[i] == nil {
						continue
					}
					_, err := tx.Exec("INSERT OR REPLACE INTO times VALUES (?, ?, ?)", buffer[i].Name, buffer[i].Topic, buffer[i].Time.Unix())
					if err != nil {
						log.Println("Can not insert access time:", err)
					}
				}
				err = tx.Commit()
				if err != nil {
					log.Println("Can not commit transaction:", err)
				}
				waitWrite.Broadcast()
				buffer = make([]*save, 0, len(buffer)*2+10)
			}

		case x := <-saveTime:
			buffer = append(buffer, &x)

		case u := <-deleteUser:
			for i := range buffer {
				if buffer[i] == nil {
					continue
				}
				if buffer[i].Name == u {
					buffer[i] = nil
				}
			}
		}
	}
}

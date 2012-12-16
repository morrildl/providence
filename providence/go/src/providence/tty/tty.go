/* Copyright © 2012 Dan Morrill
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package tty

import (
  "bufio"
  "encoding/json"
  "os"
  "time"

  "providence/common"
  "providence/log"
)

/* Reads a USB TTY looking for JSON messages from a hardware monitor and
 * injects low-level (trip and reset) eventCodes into the outgoing channel.
 * Never reads from 'incoming'; accordingly, should never be registered for
 * any message types or it will eventually deadlock when the channel buffer
 * fills.
 */
func Reader(incoming chan common.Event, outgoing chan common.Event) {
  file, err := os.Open(common.Config.Tty)
  if err != nil {
    log.Error("tty.reader", "error opening ", common.Config.Tty, ", aborting ", err)
    return
  }

  type rawEvent struct {
    Which string
    Action int
  }
  reader := bufio.NewReader(file)
  dec := json.NewDecoder(reader)
  var e rawEvent
  for {
    err := dec.Decode(&e)
    if err == nil {
      outgoing <- common.Event{Which:common.Sensors[e.Which], Action:common.EventCode(e.Action), When:time.Now()}
    } else {
      log.Warn("tty.reader", "JSON parse error from tty")
    }
  }
}

var Handler = common.Handler{Reader, make(chan common.Event, 10), map[common.EventCode]int{}} // no registrations

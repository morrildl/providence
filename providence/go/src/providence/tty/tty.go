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
  "io"
  "log"
  "net/http"
  "os"
  "strconv"
  "time"

  "providence/common"
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
    log.Fatal("Error opening ", common.Config.Tty, ", aborting ", err)
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
      log.Println("WARNING: JSON parse error on tty")
    }
  }
}

/* Test-mode low-level event injector. Has the same role as ttyReader, but
 * listens on an HTTP server, so that event can be faked locally.
 */
func MockReader(incoming chan common.Event, outgoing chan common.Event) {
  c := make(chan common.Event, 5)
  go func (c chan common.Event) {
    http.HandleFunc("/fake", func(writer http.ResponseWriter, req *http.Request) {
      err := req.ParseForm()
      if err != nil {
        log.Print("WARNING: error parsing form in TTY helper: ", err)
      } else {
        which := req.Form["w"][0]
        action, _ := strconv.Atoi(req.Form["a"][0])
        c <- common.Event{Which:common.Sensors[which], Action:common.EventCode(action), When:time.Now()}
      }
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "OK")
    })

    log.Print(http.ListenAndServe(":" + strconv.Itoa(common.Config.HttpPort + 1), nil))
  }(c)
  for {
    b := <-c
    outgoing <- b
  }
}

var Handler common.Handler

func init() {
  var f func(chan common.Event, chan common.Event)
  if common.Config.MockTty {
    f = MockReader
  } else {
    f = Reader
  }
  Handler = common.Handler{f, make(chan common.Event, 10), map[common.EventCode]int{}} // no registrations
}

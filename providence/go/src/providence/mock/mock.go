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
package mock

import (
  "io"
  "net/http"
  "strconv"
  "time"

  "providence/common"
  "providence/config"
  "providence/log"
  "providence/types"
)

/* Test-mode low-level event injector. Has the same role as ttyReader, but
 * listens on an HTTP server, so that event can be faked locally.
 */
func MockReader(incoming chan types.Event, outgoing chan types.Event) {
  c := make(chan types.Event, 5)
  go func(c chan types.Event) {
    http.HandleFunc("/fake", func(writer http.ResponseWriter, req *http.Request) {
      err := req.ParseForm()
      if err != nil {
        log.Warn("mock.reader", "error parsing form in TTY helper: ", err)
      } else {
        log.Debug("mock.reader", "form: ", req.Form)
        which := req.Form["w"][0]
        action, _ := strconv.Atoi(req.Form["a"][0])
        c <- types.Event{Which: common.SensorState[which], Action: types.EventCode(action), When: time.Now()}
      }
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "OK")
    })

    log.Error("mock.reader", "unexpected server shutdown", http.ListenAndServe(":"+strconv.Itoa(config.Server.Port+1), nil))
  }(c)
  for {
    b := <-c
    outgoing <- b
  }
}

var Handler = MockReader

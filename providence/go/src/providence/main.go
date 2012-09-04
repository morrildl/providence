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

package main
import (
  "log"

  "providence/common"
  "providence/tty"
  "providence/db"
  "providence/policy"
  "providence/gcm"
)

func main() {
  /* Stores handler function and its state and registration info. */
  type handler struct {
    f func(chan common.Event, chan common.Event)
    ch chan common.Event
    eventCodes map[common.EventCode]int
  }

  // make a struct for each handler function, mapping it to events it wants to know about
  handlers := []handler{
    handler{tty.Reader, make(chan common.Event, 10), make(map[common.EventCode]int)}, // no registrations
    handler{db.Recorder, make(chan common.Event, 10), map[common.EventCode]int{
      common.TRIP: 1, common.RESET: 1, common.AJAR: 1, common.ANOMALY: 1}},
    handler{policy.SensorMonitor, make(chan common.Event, 10), map[common.EventCode]int{common.TRIP: 1, common.RESET: 1}},
    handler{gcm.Escalator, make(chan common.Event, 10), map[common.EventCode]int{common.AJAR: 1, common.ANOMALY: 1}},
  }
  if common.Config.MockTty {
    // override tty.Reader with the mock, if so instructed
    handlers[0] = handler{tty.MockReader, make(chan common.Event, 10), make(map[common.EventCode]int)} // no registrations
  }

  // start up the handlers as goroutines
  events := make(chan common.Event, 10)
  for _, h := range handlers {
    go h.f(h.ch, events)
  }

  // loop forever, sending generated events to the listeners who want to hear them
  for {
    evt := <-events
    log.Print(evt.Which.Name + " " + evt.Description())
    for _, h := range handlers {
      _, ok := h.eventCodes[evt.Action]
      if ok {
        h.ch <- evt
      }
    }
  }
}

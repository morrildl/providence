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
  "providence/camera"
  "providence/common"
  "providence/db"
  "providence/gcm"
  "providence/gpio"
  "providence/log"
  "providence/mock"
  "providence/policy"
  "providence/tty"
)

func main() {
  /* Stores handler function and its state and registration info. */
  sensorHandler := map[string]common.Handler{"GPIO": gpio.Handler, "TTY": tty.Handler, "Mock": mock.Handler}[common.Config.SensorInterface]
  handlers := []common.Handler{sensorHandler, db.Handler, policy.Handler, gcm.Handler, camera.Handler}

  // start up the handlers as goroutines
  events := make(chan common.Event, 10)
  for _, h := range handlers {
    go h.Func(h.Chan, events)
  }

  log.Status("main.dispatcher", "running")
  log.Debug("main.dispatcher", "running in debug mode")
  // loop forever, sending generated events to the listeners who want to hear them
  for {
    evt := <-events
    log.Status("main.dispatcher", "processing event for "+evt.Which.Name+" "+evt.Description())
    for _, h := range handlers {
      _, ok := h.Events[evt.Action]
      if ok {
        h.Chan <- evt
      }
    }
  }
}

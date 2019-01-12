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
  "providence/config"
  "providence/db"
  "providence/gcm"
  "providence/gpio"
  "providence/log"
  "providence/mock"
  "providence/policy"
  "providence/tty"
  "providence/types"
)

func main() {
  /* Stores handler function and its state and registration info. */
  sensorHandler := map[string]common.Handler{"GPIO": gpio.Handler, "TTY": tty.Handler, "Mock": mock.Handler}[config.Sensor.Mode]
  handlers := []common.Handler{sensorHandler, db.Handler, policy.Handler, gcm.Handler, camera.Handler}

  // start up the handlers as goroutines
  events := make(chan types.Event, 10)
  listeners := make([]chan types.Event, len(handlers))
  for i, h := range handlers {
    listeners[i] = make(chan types.Event, 10)
    go h(listeners[i], events)
  }

  log.Status("main.dispatcher", "running")
  log.Debug("main.dispatcher", "running in debug mode")

  for {
    evt := <-events
    log.Status("main.dispatcher", "processing event for "+evt.EventID+" "+evt.Description())
    for _, l := range listeners {
      l <- evt
    }
  }
}

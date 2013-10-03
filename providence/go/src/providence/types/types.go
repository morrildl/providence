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

package types

import (
  "time"
)

type SensorType int

const (
  WINDOW SensorType = iota
  DOOR
  MOTION
)

type Sensor struct {
  Name string
  ID   string
  Kind SensorType
}

func (sensor Sensor) KindName() string {
  return map[SensorType]string{
    WINDOW: "Window",
    DOOR:   "Door",
    MOTION: "Motion Sensor",
  }[sensor.Kind]
}

type EventCode int

const (
  TRIP EventCode = iota
  RESET
  AJAR
  AJAR_RESOLVED
  ANOMALY
)

type Event struct {
  Which  Sensor
  Action EventCode
  When   time.Time
  Id     string
}

/* Returns a sensor-type-specific human string for an event code.  */
func (event Event) Description() string {
  return map[SensorType]map[EventCode]string{
    WINDOW: map[EventCode]string{
      TRIP:          "Opened",
      RESET:         "Closed",
      AJAR:          "Ajar",
      AJAR_RESOLVED: "Closed",
      ANOMALY:       "Unexpectedly Opened",
    },
    DOOR: map[EventCode]string{
      TRIP:          "Opened",
      RESET:         "Closed",
      AJAR:          "Ajar",
      AJAR_RESOLVED: "Closed",
      ANOMALY:       "Unexpectedly Opened",
    },
    MOTION: map[EventCode]string{
      TRIP:          "Detected Motion",
      RESET:         "Still",
      AJAR:          "Ongoing Motion",
      AJAR_RESOLVED: "Still",
      ANOMALY:       "Unexpected Motion",
    },
  }[event.Which.Kind][event.Action]
}

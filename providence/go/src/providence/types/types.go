/* Copyright © 2013 Dan Morrill
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
  "crypto/rand"
  "time"
)

type SensorModality int
const (
  NORMALLY_OPEN SensorModality = iota
  NORMALLY_CLOSED
  RINGING
)

type SensorSubject int
const (
  DOOR SensorSubject = iota
  WINDOW
  MOTION
)

type Event struct {
  EventID string
  SensorID string
  Trip time.Time
  Reset *time.Time
  IsAjar bool
  IsAnomalous bool
}

type Sensor struct {
  SensorID string
  Name string
  Modality SensorModality
  Subject SensorSubject
}

func (s Sensor) SubjectName() string {
  switch s.Subject {
  case DOOR:
    return "Door"
  case WINDOW:
    return "Window"
  case MOTION:
    return "Motion Sensor"
  }
}

var Sensors = make(map[string]Sensor)

func NewEvent(which string) Event {
  s, ok := Sensors[which]
  if !ok {
    return Event{}
  }
  // TODO: do something to guarantee uniqueness?
  buf := make([]byte, 16)
  io.ReadFull(rand.Reader, buf)
  return Event{
    EventID: fmt.Sprintf("%x", buf),
    Trip: time.Now(),
    SensorID: which,
  }
}

func (ev Event) Description() string {
  sensor, ok := Sensors[ev.SensorID]
  if !ok {
    log.Error("types.Event.Description", "called on event with bogus sensor '" + ev.SensorID +"'")
    return ""
  }

  var desc string
  switch {
  case ev.Reset != nil:
    if sensor.Subject == types.MOTION {
      desc = "Still"
    } else {
      desc = "Closed"
    }
  case ev.IsAjar:
    if sensor.Subject == types.MOTION {
      desc = "Motion"
    } else {
      desc = "Ajar"
    }
  default:
    if sensor.Subject == types.MOTION {
      desc = "Motion"
    } else {
      desc = "Closed"
    }
  }
  return fmt.Sprintf("%v %v", sensor.Name, desc)
}

func (ev Event) Sensor() Sensor {
  sensor, ok := Sensors[ev.SensorID]
  if !ok {
    log.Error("types.Event.Sensor()", "called on event with bogus sensor '" + ev.SensorID + "'")
    return Sensor{}
  }
  return sensor
}

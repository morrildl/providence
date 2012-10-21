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

package common

import (
  "encoding/json"
  "flag"
  "io/ioutil"
  "log"
  "os"
  "time"

  plog "providence/log"
)

type SensorType int
const (
  WINDOW SensorType = iota
  DOOR
  MOTION
)
type Sensor struct {
  Name string
  ID string
  Kind SensorType
}
func (sensor Sensor) KindName() string {
  return map[SensorType]string{
    WINDOW: "Window",
    DOOR: "Door",
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
  Which Sensor
  Action EventCode
  When time.Time
  Id string
}

/* Returns a sensor-type-specific human string for an event code.  */
func (event Event) Description() string {
  return map[SensorType]map[EventCode]string{
    WINDOW: map[EventCode]string{
      TRIP: "Opened",
      RESET: "Closed",
      AJAR: "Ajar",
      AJAR_RESOLVED: "Closed",
      ANOMALY: "Unexpectedly Opened",
    },
    DOOR: map[EventCode]string{
      TRIP: "Opened",
      RESET: "Closed",
      AJAR: "Ajar",
      AJAR_RESOLVED: "Closed",
      ANOMALY: "Unexpectedly Opened",
    },
    MOTION: map[EventCode]string{
      TRIP: "Detected Motion",
      RESET: "Still",
      AJAR: "Ongoing Motion",
      AJAR_RESOLVED: "Still",
      ANOMALY: "Unexpected Motion",
    },
  }[event.Which.Kind][event.Action]
}

var Sensors map[string]Sensor

/* Global config structure. */
var Config struct {
  Tty string
  HttpPort int
  DatabasePath string
  MockTty bool
  Debug bool
  OAuthToken string
  SensorNames map[string]string
  SensorTypes map[string]SensorType
  AjarThreshold time.Duration
  ImageRetention string
  ImageDirectory string
  ImageUrlRoot string
  CameraConfig map[string][]struct {
    Url string
    Interval int
    Count int
  }
  ExclusionIntervals []struct {
    Start string
    Duration string
    DaysOfWeek []time.Weekday // int, 0 - 6, 0 = Sunday
  }
}

func init() {
  var flagPort int
  var flagTty string
  var flagDatabase string
  var configFile string
  var mockTty bool
  var debug bool
  var oAuthToken string
  flag.StringVar(&configFile, "config", "./config.json", "fully qualified path to the JSON config file")
  flag.IntVar(&flagPort, "port", 4280, "port of the HTTP server")
  flag.StringVar(&flagTty, "tty", "/dev/ttyUSB0", "USB TTY file connected to the Arduino")
  flag.StringVar(&flagDatabase, "database", "./db.sqlite3", "fully qualified path to the sqlite3 database file")
  flag.BoolVar(&mockTty, "mocktty", false, "use an HTTP server on :4281 instead of a TTY")
  flag.BoolVar(&debug, "debug", false, "use an HTTP server on :4281 instead of a TTY")
  flag.StringVar(&oAuthToken, "oauth", "", "specify the OAuth token to send to GCM")
  flag.Parse()

  file, err := os.Open(configFile)
  if err != nil {
    log.Fatal("loading config failed opening the config file '" + configFile + "'", err)
  }
  jsonText, err := ioutil.ReadAll(file)
  if err != nil {
    log.Fatal("loading config failed reading the config file '" + configFile + "'", err)
  }
  err = json.Unmarshal([]byte(jsonText), &Config)
  if err != nil {
    log.Fatal("loading config failed on unmarshal ", err)
  }
  if flagTty != "" {
    Config.Tty = flagTty
  }
  if flagPort > 0 {
    Config.HttpPort = flagPort
  }
  if flagDatabase != "" {
    Config.DatabasePath = flagDatabase
  }
  if mockTty {
    Config.MockTty = mockTty
  }
  if debug {
    Config.Debug = debug
  }
  if oAuthToken != "" {
    Config.OAuthToken = oAuthToken
  }

  Sensors = make(map[string]Sensor)
  for k, v := range Config.SensorNames {
    Sensors[k] = Sensor{v, k, Config.SensorTypes[k]}
  }

  if Config.Debug {
    plog.SetLogLevel(plog.LEVEL_DEBUG)
  }
}

type Handler struct {
  Func func(chan Event, chan Event)
  Chan chan Event
  Events map[EventCode]int
}

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
package policy

import (
  "crypto/rand"
  "fmt"
  "io"
  "time"

  "providence/common"
  "providence/config"
  "providence/log"
  "providence/types"
)

type timeWindow struct {
  Hour     int
  Minute   int
  Duration time.Duration
  Weekdays []time.Weekday
}

/* Parse string times & durations from config into a struct. */
func parseExclusionIntervals() []timeWindow {
  windows := []timeWindow{}
  for _, w := range config.Sensor.ExclusionIntervals {
    providedStart, err := time.Parse("3:04pm", w.Start)
    if err != nil {
      log.Warn("policy.exclusions", "exclusion interval time failed to parse ", w.Start)
      continue
    }
    duration, err := time.ParseDuration(w.Duration)
    if err != nil {
      log.Warn("policy.exclusions", "exclusion interval duration failed to parse ", w.Duration)
      continue
    }
    windows = append(windows, timeWindow{
      Hour:     providedStart.Hour(),
      Minute:   providedStart.Minute(),
      Duration: duration,
      Weekdays: w.DaysOfWeek})
  }
  return windows
}

/* Generates a random ID. Does not guarantee uniqueness. */
func GetId() string {
  // TODO: probably move into db package to guarantee uniqueness
  buf := make([]byte, 16)
  io.ReadFull(rand.Reader, buf)
  return fmt.Sprintf("%x", buf)
}

/* Looks for low-level events on the incoming channel and applies some
 * heuristics to determine whether they are noteworthy. Will inject
 * higher-level eventCodes (ajar, anomalous) to the outgoing channel as
 * appropriate.
 *
 * Currently the heuristics are exclusion intervals and 'door is ajar'
 * detection. Should only be registered for low-level events.
 */
func SensorMonitor(incoming chan types.Event, outgoing chan types.Event) {
  // local structs used in synthesizing human-meaningful events from raw events
  type ajarRuleState struct {
    when     time.Time
    lastSend time.Duration
  }

  ajarThreshold := config.Sensor.AjarThreshold * time.Second
  resendFrequency := 1 * time.Minute
  lastTrips := make(map[string]*ajarRuleState)
  ticker := time.Tick(1 * time.Second)

  // pre-parse the exclusion windows and ajar rules so we don't perpetually
  // re-parse in the ticker loop
  windows := parseExclusionIntervals()
  for {
    select {
    case e := <-incoming:
      now := time.Now()

      // timestamp trips for ajar-detection, and clear on resets
      if e.Action == types.TRIP {
        lastTrips[e.Which.ID] = &ajarRuleState{now, ajarThreshold}
      } else if e.Action == types.RESET {
        delete(lastTrips, e.Which.ID)
      }

      // check trips against exclusion intervals for anomalous events
      if e.Action == types.TRIP {
        inWindow := false
        if e.Which.Kind != types.MOTION {
          // skip windows and always send motion events, as they are more
          // like state updates than events
          for _, w := range windows {
            legit := false
            for _, dow := range w.Weekdays {
              if now.Weekday() == dow {
                legit = true
                break
              }
            }
            if !legit {
              continue
            }
            start := time.Date(now.Year(), now.Month(), now.Day(), w.Hour, w.Minute, 0, 0, time.Local)
            end := start.Add(w.Duration)
            if now.After(start) && now.Before(end) {
              inWindow = true
              break
            }
          }
        }
        if !inWindow {
          outgoing <- types.Event{Which: e.Which, Action: types.ANOMALY, When: now, Id: GetId()}
        }
      }

    case <-ticker:
      // once per second, check whether anything is (still) Ajar & (re)transmit if it's time to
      for which, last := range lastTrips {
        if time.Since(last.when) > ajarThreshold && time.Since(last.when) > last.lastSend {
          last.lastSend += resendFrequency
          outgoing <- types.Event{Which: common.SensorState[which], Action: types.AJAR, When: time.Now(), Id: GetId()}
        }
      }
    }
  }
}

var Handler = common.Handler{SensorMonitor, make(chan types.Event, 10), map[types.EventCode]int{types.TRIP: 1, types.RESET: 1}}

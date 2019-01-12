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

package gcm

import (
  "bytes"
  "encoding/json"
  "io/ioutil"
  "net/http"
  "strconv"
  "time"

  "providence/common"
  "providence/config"
  "providence/db"
  "providence/log"
  "providence/server"
  "providence/types"
)

type payload struct {
  EventID          string
  EventDescription string
  EventTrip        time.Time
  IsAjar           bool
  SensorName       string
  SensorType       string
  SensorTypeName   string
}
type request struct {
  data payload
  skip []string
}

func startTransmitter() (chan request, chan db.RegIdUpdate) {
  type gcmRequest struct {
    RegistrationIds []string `json:"registration_ids"`
    Data            payload  `json:"data"`
  }
  type gcmResponse struct {
    MulticastId  uint64 `json:"multicast_id"`
    Success      int    `json:"success"`
    Failure      int    `json:"failure"`
    CanonicalIds int    `json:"canonical_ids"`
    Results      []struct {
      MessageId      string `json:"message_id"`
      RegistrationId string `json:"registration_id"`
      Error          string `json:"error"`
    } `json:"results"`
  }

  // define some constants and structs used only for JSON data formatting
  // & communication with GCM
  const (
    GCM_URL      = "https://android.googleapis.com/gcm/send"
    GCM_MIMETYPE = "application/json"
  )

  requestSource := make(chan request, 10)
  regIdUpdateSink := make(chan db.RegIdUpdate, 10)

  go func() {
    for {
      select {
      case req := <-requestSource:
        regIds, err := db.GetRegIds(req.skip)
        if err != nil {
          log.Warn("gcm.transmitter", "failed getting RegIDs during GCM send ", err)
          continue
        }
        if len(regIds) == 0 {
          // no recipients == nothing to do
          continue
        }

        // format up a GCM JSON message for the request
        j, ok := json.Marshal(gcmRequest{regIds, req.data})
        log.Debug("gcm.transmitter", "DEBUG: GCM request as follows:")
        log.Debug("gcm.transmitter", string(j))
        if ok != nil {
          log.Status("gcm.transmitter", "JSON failure during encode for GCM", ok)
          break
        }

        // send the event to GCM server via HTTP POST, per spec
        httpReq, err := http.NewRequest("POST", GCM_URL, bytes.NewReader(j))
        if err != nil {
          log.Error("gcm.transmitter", "Failed to create GCM HTTP request", err)
          break
        }
        httpReq.Header.Add("Authorization", "key="+config.GCM.OAuthToken)
        httpReq.Header.Add("Content-Type", GCM_MIMETYPE)
        client := &http.Client{}
        httpResp, err := client.Do(httpReq)
        if err != nil {
          log.Warn("gcm.transmitter", "GCM request failed during execution", err)
          break
        }
        defer httpResp.Body.Close()

        // look at the JSON response from GCM server & take any actions indicated
        body, err := ioutil.ReadAll(httpResp.Body)
        if err == nil && len(body) > 0 {
          log.Debug("gcm.transmitter", "GCM response payload as follows:")
          log.Debug("gcm.transmitter", string(body))
          var jsonResponse gcmResponse
          jsonErr := json.Unmarshal(body, &jsonResponse)
          if jsonErr != nil {
            log.Error("gcm.transmitter", "JSON unmarshal failure on GCM response: ", jsonErr)
            log.Error("gcm.transmitter", string(body))
            break
          }
          log.Status("gcm.transmitter", "GCM response summary: success: ", jsonResponse.Success, "; failure: ", jsonResponse.Failure)
          for i, oldId := range regIds {
            result := jsonResponse.Results[i]

            // GCM server sent a "canonical registration ID" message; update our list accordingly
            if result.RegistrationId != "" {
              regIdUpdateSink <- db.RegIdUpdate{oldId, result.RegistrationId, false}
            }

            // check to see if the reg ID had a permanent error, and if so remove from the list
            if result.Error != "" && result.Error != "Unavailable" {
              regIdUpdateSink <- db.RegIdUpdate{oldId, "", true}
            }
          }
        }
      } // select
    } //for
  }()

  return requestSource, regIdUpdateSink
}

/* Watches for higher-level event types and escalates them for
 * human review -- i.e. via GCM. Should only be registered for AJAR and
 * ANOMALY.
 */
func Escalator(incoming chan types.Event, outgoing chan types.Event) {
  regIdUpdateSink := db.StartRegIdUpdater()

  // start the HTTP server which is our source for regID creates & deletes
  regIdHttpSource, gcmRequestSource := server.Start()

  // start the GCM helper
  gcmRequestSink, regIdGcmUpdateSource := startTransmitter()

  // check each raw event and synthesize higher level events as appropriate
  for {
    select {
    // the HTTP server feeds us RegIDs from new clients, & the transmitter can
    // feed us deletes of and updates to existing RegIDs
    case regIdUpdate := <-regIdHttpSource:
      regIdUpdateSink <- regIdUpdate
    case regIdUpdate := <-regIdGcmUpdateSource:
      regIdUpdateSink <- regIdUpdate

    // New URL share request
    case urlRequest := <-gcmRequestSource:
      gcmRequestSink <- request{payload{Url: urlRequest.Url}, urlRequest.Skip}

    // New monitoring event from the dispatcher.
    case ev := <-incoming:
      if !ev.IsAjar && !ev.IsAnomalous {
        log.Debug("gcm.Escalator", "skipping mundane event '" + ev.EventID + "'")
        break
      }
      sensor := ev.Sensor()
      gcmRequestSink <- request{
        payload{ // GCM only supports strings so we can't be very typesafe here
          ev.EventID, ev.Description(), ev.Trip, ev.IsAjar, sensor.Name,
          strconv.Itoa(int(sensor.Subject)), sensor.SubjectName()))},
        []string{},
      }
    }
  }
}

var Handler common.Handler = Escalator

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

package camera

import (
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strconv"
  "time"

  "providence/common"
)

func captureImage(url string, ids []string) {
  s := time.Now().UnixNano()

  res, err := http.Get(url)
  if err != nil {
    log.Print("WARNING: failed to get image URL " + url)
    log.Print("WARNING: reason was ", err)
    return
  }
  defer res.Body.Close()

  body, err := ioutil.ReadAll(res.Body)
  if err != nil {
    log.Print("WARNING: failed reading HTTP response for " + url)
    log.Print("WARNING: reason was ", err)
    return
  }

  r := time.Now().UnixNano()

  for _, id := range ids {
    fname := common.Config.ImageDirectory + "/" + id + "-" + strconv.FormatInt(time.Now().Unix(), 10) + ".jpg"
    file, err := os.Create(fname)
    if err != nil {
      log.Print("WARNING: failed writing image contents for " + id)
      log.Print("WARNING: url was " + url)
      log.Print("WARNING: reason was ", err)
      return
    }
    defer file.Close()
    file.Write(body)

    if common.Config.Debug {
      log.Print("wrote photo for " + id + " HTTP time:" + strconv.FormatInt(r - s, 10))
    }
  }
}

func Monitor(incoming chan common.Event, outgoing chan common.Event) {
  type configTracker struct {
    which string
    id string
    url string
    interval int
    count int
    next int
  }
  pending := make([]configTracker, 0)

  ticker := time.Tick(1 * time.Second)
  // check each raw event and synthesize higher level events as appropriate
  for {
    select {
    // one second has passed...
    case <- ticker:
      worklist := make(map[string][]string)
      if common.Config.Debug && len(pending) > 0 {
        log.Print("pending: ", pending)
      }
      old := pending
      pending = make([]configTracker, 0)
      for _, p := range old {
        p.next -= 1
        if p.next < 1 {
          ids, ok := worklist[p.url]
          if !ok {
            ids = make([]string, 0)
          }
          worklist[p.url] = append(ids, p.id)
          p.count -= 1
          p.next = p.interval
        }
        if p.count >= 1 {
          pending = append(pending, p)
        }
      }
      for url, ids := range worklist {
          go captureImage(url, ids)
      }

    // New monitoring event from the dispatcher.
    case ev := <-incoming:
      log.Print(ev)
      configs, ok := cameraConfigs[ev.Which.ID]
      if ok {
        for _, config := range configs {
          pending = append(pending, configTracker{ev.Which.ID, ev.Id, config.Url, config.Interval, config.Count, config.Interval})
        }
      } /* else { } // ok == false is fine, it just means no camera is configured for that sensor */
    }
  }
}

type cameraConfig struct {
  Url string
  Interval int
  Count int
}
var cameraConfigs map[string][]cameraConfig

func init() {
  cameraConfigs = make(map[string][]cameraConfig)
  for which, configs := range common.Config.CameraConfig {
    cameraConfigs[which] = make([]cameraConfig, 0)
    for _, config := range configs {
      cameraConfigs[which] = append(cameraConfigs[which], config)
    }
  }
}

var Handler = common.Handler{Monitor, make(chan common.Event, 10), map[common.EventCode]int{common.AJAR: 1, common.ANOMALY: 1}}

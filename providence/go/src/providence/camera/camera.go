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
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"
  "time"

  "providence/common"
  "providence/config"
  "providence/log"
  "providence/types"
)

/* A goroutine that runs once an hour and purges any files older than the
 * specified retention period. */
func startPhotoPurger() {
  ticker := time.Tick(1 * time.Hour)
  if config.General.Debug {
    ticker = time.Tick(1 * time.Minute)
  }
  retention, err := time.ParseDuration(config.Photo.Retention)
  if err != nil {
    log.Error("camera.purger", "bogus image retention duration '"+config.Photo.Retention+"'. Aborting.")
    return
  }
  go func() {
    for {
      select {
      case <-ticker:
        cutoff := time.Now().Add(-retention)
        log.Status("camera.purger", "purging images before "+cutoff.Format("Mon Jan 2 15:04:05 -0700 MST 2006"))
        imageDir, err := os.Open(config.Photo.Directory)
        if err != nil {
          log.Warn("camera.purger", "file open failed on "+config.Photo.Directory)
          break
        }
        finfos, err := imageDir.Readdir(-1)
        if err != nil {
          log.Warn("camera.purger", "WARNING: ReadDir failure on "+config.Photo.Directory)
          imageDir.Close()
          break
        }
        count := 0
        for _, finfo := range finfos {
          if finfo.IsDir() || !(strings.HasSuffix(finfo.Name(), ".jpg") || strings.HasSuffix(finfo.Name(), ".jpeg")) {
            continue
          }
          if cutoff.After(finfo.ModTime()) {
            err := os.Remove(filepath.Join(config.Photo.Directory, finfo.Name()))
            count += 1
            if err != nil {
              log.Error("camera.purger", "failure to remove "+finfo.Name())
            } else {
              log.Debug("camera.purger", "removed "+finfo.Name())
            }
          }
        }
        log.Status("camera.purger", "removed "+strconv.Itoa(count)+" images")

        imageDir.Close()
      }
    }
  }()
}

/* Fetches the indicated URL and saves the response body on behalf of the
 * indicated IDs. Does not actually verify that the response is JPEG data, but
 * always names the files that way. */
func captureImage(url string, ids []string) {
  s := time.Now().UnixNano()

  res, err := http.Get(url)
  if err != nil {
    log.Warn("camera.capture", "to get image URL "+url)
    log.Warn("camera.capture", "reason was ", err)
    return
  }
  defer res.Body.Close()

  body, err := ioutil.ReadAll(res.Body)
  if err != nil {
    log.Warn("camera.capture", "failed reading HTTP response for "+url)
    log.Warn("camera.capture", "reason was ", err)
    return
  }

  r := time.Now().UnixNano()

  for _, id := range ids {
    fname := filepath.Join(config.Photo.Directory, id+"-"+time.Now().Format("20060102150405.00")+".jpg")
    file, err := os.Create(fname)
    if err != nil {
      log.Warn("camera.capture", "failed writing image contents for "+id)
      log.Warn("camera.capture", "url was "+url)
      log.Warn("camera.capture", "reason was ", err)
      return
    }
    defer file.Close()
    file.Write(body)

    log.Debug("camera.capture", "wrote photo for "+id+" HTTP time:"+strconv.FormatInt(r-s, 10))
  }
}

/* Handler for main.go. */
func Monitor(incoming chan types.Event, outgoing chan types.Event) {
  type configTracker struct {
    which    string
    id       string
    url      string
    interval int
    count    int
    next     int
  }
  pending := make([]configTracker, 0)

  ticker := time.Tick(1 * time.Second)
  // check each raw event and synthesize higher level events as appropriate
  for {
    select {
    // grab any requested URLs, snapping everything to once per second
    case <-ticker:
      worklist := make(map[string][]string)
      if len(pending) > 0 {
        log.Debug("camera.handler", "pending: ", pending)
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
      if !ev.IsAnomalous {
        log.Debug("camera.handler", "skipping mundane event '" + ev.EventID "'")
        break
      }
      log.Debug("camera.handler", "processing event ", ev)
      configs, ok := cameraConfigs[ev.SensorID]
      if ok {
        for _, cfg := range configs {
          pending = append(pending, configTracker{ev.SensorID, ev.EventID, cfg.Url, cfg.Interval, cfg.Count, cfg.Interval})
        }
      } /* else { } // ok == false is fine, it just means no camera is configured for that sensor */
    }
  }
}

var cameraConfigs map[string][]config.CameraSpecConfig

func init() {
  // pre-parse the camera configuration structure; basically just says how
  // many photos to grab from what URL at what interval, for any event from a
  // given sensor
  cameraConfigs = make(map[string][]config.CameraSpecConfig)
  for which, configs := range config.Photo.CameraSpec {
    cameraConfigs[which] = make([]config.CameraSpecConfig, 0)
    for _, cfg := range configs {
      cameraConfigs[which] = append(cameraConfigs[which], cfg)
    }
  }

  startPhotoPurger()
}

var Handler common.Handler = Monitor

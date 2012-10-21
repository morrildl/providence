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
package server

import (
  "encoding/json"
  "io"
  "io/ioutil"
  "log"
  "net/http"
  "net/url"
  "os"
  "path/filepath"
  "strconv"
  "strings"

  "providence/common"
  "providence/db"
)

type ShareUrlRequest struct {
  Url string
  Skip []string
}
/* Spins up an HTTP server in a goroutine to which user devices make requests
 * to add & delete registration IDs, per the GCM spec. Server also implements
 * a trivial heartbeat URL that devices can use to detect if the monitor goes
 * offline, and notify locally.
 */
func Start() (chan db.RegIdUpdate, chan ShareUrlRequest) {
  regIdRequestChan := make(chan db.RegIdUpdate, 5)
  gcmSendUrlChan := make(chan ShareUrlRequest, 5)
  go func () {
    // registration ID handler; RESTful:
    // - POST = add reg ID(s) listed in body
    // - DELETE = discard reg ID(s) listed in body
    http.HandleFunc("/regid", func(writer http.ResponseWriter, req *http.Request) {
      body, err := ioutil.ReadAll(req.Body)
      if err != nil {
        log.Print("HTTP request read failure", err)
      } else {
        log.Print("/regid: ", req.Method)
        remove := req.Method == "DELETE"
        for _, s := range strings.Split(string(body), "\n") {
          regIdRequestChan <- db.RegIdUpdate{s, "", remove}
        }
      }
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "OK\n")
    })

    // heartbeat URL: always returns code=200 with message of "HI"
    http.HandleFunc("/heartbeat", func(writer http.ResponseWriter, req *http.Request) {
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "HI\n")
    })

    // Send a VBOF URL to subscribers
    http.HandleFunc("/vbof", func(writer http.ResponseWriter, req *http.Request) {
      body, err := ioutil.ReadAll(req.Body)
      if err != nil {
        log.Print("WARNING: failure reading body in /vbof", err)
      } else {
        bodyStr := string(body)
        if (bodyStr == "") {
          log.Print("WARNING: empty body in /vbof")
        } else {
          bodyStr, _ = url.QueryUnescape(bodyStr)
          lines := strings.Split(bodyStr, "\n")
          if len(lines) < 1 || len(lines) > 2 {
            log.Print("WARNING: unexpected number of lines in /vbof", lines)
          } else {
            uri := lines[0]
            _, err := url.Parse(uri)
            if err != nil {
              log.Print("WARNING: input does not look like a URL", uri)
            } else {
              skip := make([]string, 0)
              if len(lines) > 1 {
                skip = append(skip, lines[1])
              }
              gcmSendUrlChan <- ShareUrlRequest{lines[0], skip}
            }
          }
        }
      }
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "OK\n")
    })

    // a way for an app to query a list of photo URLs for a given ID
    // The ID will have been sent to the app via GCM; this is how it pulls
    // photos, if any. This returns only the list, it does NOT return JPEG
    // data.
    http.HandleFunc("/photos", func(writer http.ResponseWriter, req *http.Request) {
      doerr := func() {
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
      }

      body, err := ioutil.ReadAll(req.Body)
      if err != nil {
        log.Print("WARNING: failure reading body in /photos", err)
        doerr()
        return
      }

      dir, err := os.Open(common.Config.ImageDirectory)
      if err != nil {
        log.Print("ERROR: failed to open " + common.Config.ImageDirectory)
        doerr()
        return
      }
      defer dir.Close()
      finfos, err := dir.Readdir(-1)
      if err != nil {
        log.Print("ERROR: failed to enumerate " + common.Config.ImageDirectory)
        doerr()
        return
      }

      imagesById := make(map[string][]string)
      for _, finfo := range finfos {
        if finfo.IsDir() || !(strings.HasSuffix(finfo.Name(), ".jpg") || strings.HasSuffix(finfo.Name(), ".jpeg")) {
          continue
        }
        split := strings.SplitN(finfo.Name(), "-", 2)
        images, ok := imagesById[split[0]]
        if !ok {
          images = make([]string, 0)
        }
        imagesById[split[0]] = append(images, finfo.Name())
      }

      urlsById := make(map[string][]string)
      for _, id := range strings.Split(string(body), "\n") {
        if len(id) == 0 {
          continue
        }
        files, ok := imagesById[id]
        if !ok || len(files) < 1 {
          continue
        }
        urls := make([]string, 0)
        for _, file := range files {
          urls = append(urls, common.Config.ImageUrlRoot + file)
        }
        urlsById[id] = urls
      }

      bodyStr, err := json.Marshal(urlsById)
      if err != nil {
        log.Print("ERROR: could not marshal to JSON")
        log.Print(urlsById)
        doerr()
        return
      }
      if common.Config.Debug {
        log.Print("Marshaled JSON:")
        log.Print(string(bodyStr))
      }

      writer.Header().Add("Content-Type", "application/json")
      writer.Header().Add("Content-Length", strconv.Itoa(len(bodyStr)))
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, string(bodyStr))
    })

    // fetch and return an indicated photo
    http.HandleFunc("/photo/", func(writer http.ResponseWriter, req *http.Request) {
      fnames := strings.Split(req.URL.Path, "/")
      if len(fnames) != 3 {
        // means there is one or more extra chunks in there, which could be an attack; do nothing
        writer.WriteHeader(http.StatusNotFound)
        return
      }
      fname := fnames[len(fnames) - 1]
      fpath := filepath.Join(common.Config.ImageDirectory, fname)
      f, err := os.Open(fpath)
      if err != nil {
        writer.WriteHeader(http.StatusNotFound)
        return
      }
      defer f.Close()
      bytes, err := ioutil.ReadAll(f)
      if err != nil {
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
        return
      }
      // TODO: add authentication
      log.Print("Serving " + fname + " to " + req.RemoteAddr)
      writer.Header().Add("Content-Type", "image/jpeg")
      writer.Header().Add("Content-Length", strconv.Itoa(len(bytes)))
      io.WriteString(writer, string(bytes))
    })

    // listen on the configured port
    log.Print(http.ListenAndServe(":" + strconv.Itoa(common.Config.HttpPort), nil))
  }()
  return regIdRequestChan, gcmSendUrlChan
}

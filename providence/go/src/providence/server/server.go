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
  "net/http"
  "os"
  "path/filepath"
  "strconv"
  "strings"

  "providence/common"
  "providence/db"
  "providence/log"
  "providence/policy"
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
        log.Warn("server.RegID", "HTTP request read failure", err)
      } else {
        log.Status("server.RegID", "/regid: ", req.Method)
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
    http.HandleFunc("/vbofupload", func(writer http.ResponseWriter, req *http.Request) {
      log.Debug("server.vbof", "incoming request from " + req.RemoteAddr)
      body, err := ioutil.ReadAll(req.Body)
      if err != nil || len(body) < 1 {
        log.Warn("server.vbof", "failure reading body in /vbofupload", err)
        writer.WriteHeader(http.StatusBadRequest)
        io.WriteString(writer, "FAIL")
        return
      }

      mimeType := req.Header.Get("Content-Type")
      if mimeType == "" {
        log.Warn("server.vbof", "no content-type in /vbofupload", err)
        writer.WriteHeader(http.StatusBadRequest)
        io.WriteString(writer, "FAIL")
        return
      }
      extension := map[string]string{
        "image/jpeg": "jpg",
        "image/png": "png",
        "image/gif": "gif",
        "image/tiff": "tif",
        "image/*": "jpg", // FAIL
      }[mimeType]

      req.ParseForm()
      title := req.Form.Get("subject") // don't care if "" as text is optional

      vbof_id := policy.GetId()

      err = db.StoreVbofInfo(vbof_id, mimeType, title)
      if err != nil {
        log.Warn("server.vbof", "no failed storing metadata", err)
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
        return
      }

      fname := filepath.Join(common.Config.VbofImageDirectory, vbof_id + "." + extension)
      file, err := os.Create(fname)
      if err != nil {
        log.Warn("camera.capture", "failed writing image contents for new vbof ", err)
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
        return
      }
      defer file.Close()
      file.Write(body)

      url := common.Config.VbofImageUrlRoot + vbof_id + "." + extension
      log.Debug("server.vbof", req.RemoteAddr + " " + url)
      gcmSendUrlChan <- ShareUrlRequest{url, make([]string, 0)}

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
        log.Warn("server.photos", "failure reading body in /photos", err)
        doerr()
        return
      }

      dir, err := os.Open(common.Config.ImageDirectory)
      if err != nil {
        log.Error("server.photos", "failed to open " + common.Config.ImageDirectory)
        doerr()
        return
      }
      defer dir.Close()
      finfos, err := dir.Readdir(-1)
      if err != nil {
        log.Error("server.photos", "failed to enumerate " + common.Config.ImageDirectory)
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
        log.Error("server.photos", "could not marshal to JSON")
        log.Error("server.photos", urlsById)
        doerr()
        return
      }
      log.Debug("server.photos", "marshaled JSON:")
      log.Debug("server.photos", string(bodyStr))

      writer.Header().Add("Content-Type", "application/json")
      writer.Header().Add("Content-Length", strconv.Itoa(len(bodyStr)))
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, string(bodyStr))
    })

    serve_image := func(path string, message string, mimeType string, writer http.ResponseWriter, req *http.Request) {
      f, err := os.Open(path)
      if err != nil {
        log.Debug("server.photo", "404 URL " + req.URL.Path + " from " + req.RemoteAddr)
        writer.WriteHeader(http.StatusNotFound)
        io.WriteString(writer, "404")
        return
      }
      defer f.Close()

      bytes, err := ioutil.ReadAll(f)
      if err != nil {
        log.Error("server.photo", "failed reading file for URL " + req.URL.Path + " from " + req.RemoteAddr)
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
        return
      }

      log.Status("server.photo", "serving " + path + " to " + req.RemoteAddr)

      writer.Header().Add("Content-Type", mimeType)
      if message != "" {
        writer.Header().Add("X-Image-Title", message)
      }
      writer.Header().Add("Content-Length", strconv.Itoa(len(bytes)))
      io.WriteString(writer, string(bytes))
    }

    http.HandleFunc("/vbof/", func(writer http.ResponseWriter, req *http.Request) {
      fnames := strings.Split(req.URL.Path, "/")
      if len(fnames) != 3 {
        // means there is one or more extra chunks in there, which could be an attack; do nothing
        log.Warn("server.vbof", "nonconformant URL " + req.URL.Path + " from " + req.RemoteAddr)
        writer.WriteHeader(http.StatusNotFound)
        io.WriteString(writer, "404")
        return
      }
      fname := fnames[len(fnames) - 1]
      fpath := filepath.Join(common.Config.VbofImageDirectory, fname)

      vbof_id := strings.Split(fname, ".")[0]
      mimeType, title, err := db.GetVbofInfo(vbof_id)
      if err != nil || mimeType == "" {
        log.Debug("server.vbof", mimeType + " " + title, err)
        log.Warn("server.vbof", "request for nonexistent VBOF " + vbof_id, err)
        writer.WriteHeader(http.StatusNotFound)
        io.WriteString(writer, "404")
        return
      }

      serve_image(fpath, title, mimeType, writer, req)
    })

    // fetch and return an indicated photo
    http.HandleFunc("/photo/", func(writer http.ResponseWriter, req *http.Request) {
      fnames := strings.Split(req.URL.Path, "/")
      if len(fnames) != 3 {
        // means there is one or more extra chunks in there, which could be an attack; do nothing
        log.Warn("server.photo", "nonconformant URL " + req.URL.Path + " from " + req.RemoteAddr)
        writer.WriteHeader(http.StatusNotFound)
        io.WriteString(writer, "404")
        return
      }
      fname := fnames[len(fnames) - 1]
      fpath := filepath.Join(common.Config.ImageDirectory, fname)

      serve_image(fpath, "", "image/jpeg", writer, req)
    })

    // listen on the configured port
    port := strconv.Itoa(common.Config.ServerPort)
    haveCert := common.Config.HttpsCertFile != "" && common.Config.HttpsKeyFile != ""
    if haveCert {
      log.Status("server.http", "starting HTTPS on port " + port)
      log.Error("server.http", "shut down unexpectedly",
      http.ListenAndServeTLS(":" + port, common.Config.HttpsCertFile, common.Config.HttpsKeyFile, nil))
    } else {
      log.Status("server.http", "starting HTTP on port " + port)
      log.Error("server.http", "shut down unexpectedly", http.ListenAndServe(":" + port, nil))
    }
  }()
  return regIdRequestChan, gcmSendUrlChan
}

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
  "errors"
  "fmt"
  "io"
  "io/ioutil"
  "net/http"
  "os"
  "path/filepath"
  "sort"
  "strconv"
  "strings"
  "time"

  jwt "github.com/morrildl/jwt-go"

  "providence/config"
  "providence/db"
  "providence/log"
)

var validCerts map[string]string

/* Updates an in-memory copy of the currently-published Google OAuth2 certs on
 * an hourly basis. This is used by verifyToken to determine whether a token
 * is signed by a currently active cert. Google rotates these certs fairly
 * aggressively; this makes sure we keep up. */
func startCertFetcher() {
  ticker := time.Tick(1 * time.Hour)
  go func() {
    for {
      log.Debug("server.certFetcher", "Updating validCerts map")

      res, err := http.Get(config.UserAuth.GoogleOAuthCertsURL)
      if err == nil {
        dec := json.NewDecoder(res.Body)
        err = dec.Decode(&validCerts)
        log.Debug("server.certFetcher", validCerts)

        res.Body.Close()
      } else {
        log.Warn("server.certFetcher", "failure fetching GoogleOAuthCertsURL", err)
      }

      <-ticker
    }
  }()
}

/* Inspects the indicated JWT token string to see if it is correctly constructed 
 * for Google's OAuth relying party system for Android apps. That is:
 * - the token must be properly formed
 * - it must be signed by Google, using one of the current published certs (see startCertFetcher)
 * - its 'email' claim must identify it as for a Google account whitelisted in config.Server.UserAuth
 * - its 'cid' claim must match the client ID generated by the Google API console for this app
 * - its 'aud' claim must match the audience (i.e. OAuth scope) generated by the Google API console for this app
 * Returns the email address from the claim on success; returns "" and an error on failure.
 */
func verifyToken(rawToken string) (string, error) {
  parsedToken, err := jwt.Parse(rawToken, func(token *jwt.Token) ([]byte, error) {
    kid := token.Header["kid"].(string)
    cert, ok := validCerts[kid]
    if ok {
      return []byte(cert), nil
    } else {
      return nil, errors.New("unknown kid " + kid)
    }
    return nil, errors.New("wtf")
  })

  if err != nil {
    return "", err
  }
  if !parsedToken.Valid {
    // in the jwt-go impl, .Valid := err != nil, so technically this is redundant; but check 
    // anyway for future proofing since the API doesn't actually guarantee this
    return "", errors.New("invalid signature")
  }

  log.Debug("server.verifyToken", "validated signature for token")

  authorized := false
  for _, email := range config.UserAuth.GoogleAccountWhitelist {
    if email == parsedToken.Claims["email"] {
      authorized = true
      break
    }
  }
  email := parsedToken.Claims["email"].(string)
  aud := parsedToken.Claims["aud"].(string)
  cid := parsedToken.Claims["cid"].(string)
  if !authorized {
    log.Warn("server.verifyToken", "received token from unauthorized GAIA "+email)
    return "", errors.New("token is for unauthorized " + email)
  }

  log.Debug("server.verifyToken", "aud as received = "+aud)
  log.Debug("server.verifyToken", "aud as expected = "+config.UserAuth.OAuthAudience)
  log.Debug("server.verifyToken", "cid as received = "+cid)
  log.Debug("server.verifyToken", "cid as expected = "+config.UserAuth.OAuthClientID)

  if parsedToken.Claims["aud"] != config.UserAuth.OAuthAudience {
    log.Warn("server.verifyToken", "received token for wrong aud "+aud)
    return "", errors.New("token is for unrecognized aud " + aud)
  }
  if parsedToken.Claims["cid"] != config.UserAuth.OAuthClientID {
    log.Warn("server.verifyToken", "received token for wrong cid "+cid)
    return "", errors.New("token is for unrecognized cid " + cid)
  }

  return email, nil
}

/* Checks for a legit JWT signed by Google. If the token is present and legit,
 * user is authenticated and this method returns true. If the token is
 * missing, corrupt, or for an unauthorized user (i.e. if verifyToken()
 * fails), returns false AND writes a 403 response to the request. IOW callers
 * should return early if this method returns false. */
func checkAuth(writer http.ResponseWriter, req *http.Request) bool {
  token := req.Header.Get("X-OAuth-JWT")
  if token == "" {
    log.Warn("server.checkAuth", "auth token not present")
    writer.WriteHeader(http.StatusForbidden)
    io.WriteString(writer, "NO\n")
    return false
  }
  email, err := verifyToken(token)
  if err != nil {
    log.Warn("server.checkAuth", "auth token is invalid")
    log.Warn("server.checkAuth", err)
    log.Debug("server.checkAuth", token)
    writer.WriteHeader(http.StatusForbidden)
    io.WriteString(writer, "NO\n")
    return false
  }

  log.Debug("server.checkAuth", "authenticated HTTP request from "+email)
  return true
}

type ShareUrlRequest struct {
  Url  string
  Skip []string
}

/* Spins up an HTTP server in a goroutine to which user devices make requests
 * to add & delete registration IDs, per the GCM spec. Server also implements
 * a trivial heartbeat URL that devices can use to detect if the monitor goes
 * offline, and notify locally.
 */
func Start() (chan db.RegIdUpdate, chan ShareUrlRequest) {
  startCertFetcher()
  regIdRequestChan := make(chan db.RegIdUpdate, 5)
  gcmSendUrlChan := make(chan ShareUrlRequest, 5)
  go func() {
    // registration ID handler; RESTful:
    // - POST = add reg ID(s) listed in body
    // - DELETE = discard reg ID(s) listed in body
    http.HandleFunc(config.URLPath.RegID, func(writer http.ResponseWriter, req *http.Request) {
      log.Debug("server", "incoming request to /regid")

      if !checkAuth(writer, req) {
        return
      }

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
    http.HandleFunc(config.URLPath.Heartbeat, func(writer http.ResponseWriter, req *http.Request) {
      // unauthenticated; no call to checkAuth()
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "HI\n")
    })

    // setup URL: displays a QR code that stores config info; client can scan it to set up
    http.HandleFunc(config.URLPath.QRConfig, func(writer http.ResponseWriter, req *http.Request) {
      // unauthenticated; no call to checkAuth() -- this is our bootstrap
      url, err := config.GetClientConfigQR()
      if err != nil {
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
        return
      }
      writer.WriteHeader(http.StatusOK)
      writer.Header().Add("Content-Type", "text/html")
      body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta http-equiv="refresh" content="0; url=%v"></head>
<body><script language="JavaScript">window.location="%v";</script></body>
</html>`, url, url)
      writer.Header().Add("Content-Length", strconv.Itoa(len(body)))
      io.WriteString(writer, body)
    })

    // return a list of the most recent 10 entries; intended for
    // new clients to get initial state
    http.HandleFunc(config.URLPath.Recent, func(writer http.ResponseWriter, req *http.Request) {
      if !checkAuth(writer, req) {
        return
      }

      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "HI\n")
    })

    // a way for an app to query a list of photo URLs for a given ID
    // The ID will have been sent to the app via GCM; this is how it pulls
    // photos, if any. This returns only the list, it does NOT return JPEG
    // data.
    http.HandleFunc(config.URLPath.PhotoList, func(writer http.ResponseWriter, req *http.Request) {
      if !checkAuth(writer, req) {
        return
      }

      doerr := func() {
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
      }

      log.Debug("server.photos", "servicing request for "+req.URL.Path)

      var photoIDs string
      chunks := strings.SplitN(req.URL.Path, "/", 3)[2:]
      if len(chunks) > 0 && chunks[0] != "" {
        photoIDs = chunks[0]
      } else {
        body, err := ioutil.ReadAll(req.Body)
        if err != nil {
          log.Warn("server.photos", "failure reading body in /photos", err)
          doerr()
          return
        }
        photoIDs = string(body)
      }
      if photoIDs == "" {
        log.Warn("server.photos", "could not get IDs from path or body")
        doerr()
        return
      }

      dir, err := os.Open(config.Photo.Directory)
      if err != nil {
        log.Error("server.photos", "failed to open "+config.Photo.Directory)
        doerr()
        return
      }
      defer dir.Close()
      finfos, err := dir.Readdir(-1)
      if err != nil {
        log.Error("server.photos", "failed to enumerate "+config.Photo.Directory)
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
      for _, id := range strings.Split(photoIDs, "\n") {
        if len(id) == 0 {
          continue
        }
        files, ok := imagesById[id]
        if !ok || len(files) < 1 {
          continue
        }
        urls := make([]string, 0)
        for _, file := range files {
          urls = append(urls, config.URLJoin(config.GetURLFor(config.PATH_PHOTO), file))
        }
        urlsById[id] = urls
      }

      for _, urls := range urlsById {
        sort.Strings(urls)
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
      writer.Header().Add("Cache-control", "private,max-age=7776000")
      writer.Header().Add("Expires", time.Now().Add(time.Hour*24*90).Format(time.RFC1123))
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, string(bodyStr))
    })

    serve_image := func(path string, message string, mimeType string, writer http.ResponseWriter, req *http.Request) {
      f, err := os.Open(path)
      if err != nil {
        log.Debug("server.photo", "404 URL "+req.URL.Path+" from "+req.RemoteAddr)
        writer.WriteHeader(http.StatusNotFound)
        io.WriteString(writer, "404")
        return
      }
      defer f.Close()

      bytes, err := ioutil.ReadAll(f)
      if err != nil {
        log.Error("server.photo", "failed reading file for URL "+req.URL.Path+" from "+req.RemoteAddr)
        writer.WriteHeader(http.StatusInternalServerError)
        io.WriteString(writer, "FAIL")
        return
      }

      log.Status("server.photo", "serving "+path+" to "+req.RemoteAddr)

      writer.Header().Add("Content-Type", mimeType)
      if message != "" {
        writer.Header().Add("X-Image-Title", message)
      }
      writer.Header().Add("Content-Length", strconv.Itoa(len(bytes)))
      writer.Header().Add("Cache-control", "private,max-age=7776000")
      writer.Header().Add("Expires", time.Now().Add(time.Hour*24*90).Format(time.RFC1123))
      io.WriteString(writer, string(bytes))
    }

    // fetch and return an indicated photo
    http.HandleFunc(config.URLPath.PhotoFetch, func(writer http.ResponseWriter, req *http.Request) {
      if !checkAuth(writer, req) {
        return
      }
      log.Debug("server.photo", "request method: "+req.Method)

      fnames := strings.Split(req.URL.Path, "/")
      if len(fnames) != 3 {
        // means there is one or more extra chunks in there, which could be an attack; do nothing
        log.Warn("server.photo", "nonconformant URL "+req.URL.Path+" from "+req.RemoteAddr)
        writer.WriteHeader(http.StatusNotFound)
        io.WriteString(writer, "404")
        return
      }
      fname := fnames[len(fnames)-1]
      fpath := filepath.Join(config.Photo.Directory, fname)

      serve_image(fpath, "", "image/jpeg", writer, req)
    })

    // listen on the configured port
    port := strconv.Itoa(config.Server.Port)
    haveCert := config.Server.HttpsCertFile != "" && config.Server.HttpsKeyFile != ""
    if haveCert {
      log.Status("server.http", "starting HTTPS on port "+port)
      log.Error("server.http", "shut down unexpectedly",
        http.ListenAndServeTLS(":"+port, config.Server.HttpsCertFile, config.Server.HttpsKeyFile, nil))
    } else {
      log.Status("server.http", "starting HTTP on port "+port)
      log.Error("server.http", "shut down unexpectedly", http.ListenAndServe(":"+port, nil))
    }
  }()
  return regIdRequestChan, gcmSendUrlChan
}

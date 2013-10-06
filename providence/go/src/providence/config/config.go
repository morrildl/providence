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

package config

/*
 * Contains general configuration data for the whole server. Config data is
 * specified in a JSON file which is loaded by this module and used to
 * populate the various config objects. There are multiple such objects so
 * that code can 'import ( "providence/config" )' and then say things like
 * 'if config.General.Debug', etc.
 */

import (
  "encoding/base64"
  "encoding/json"
  "flag"
  "fmt"
  "io/ioutil"
  "log"
  "os"
  "strconv"
  "strings"
  "time"
  "net/url"

  "providence/common"
  plog "providence/log"
  "providence/types"
)

type GeneralConfig struct {
  Debug        bool
  DatabasePath string
  LogFile      string
  QRGenURL     string
}

var General = GeneralConfig{
  Debug:        false,
  DatabasePath: "./providence.sqlite3",
  LogFile:      "./providence.log",
  QRGenURL:     "http://qrfree.kaywa.com/?l=1&s=8&d=",
}

type ServerConfig struct {
  Port          int
  URLRoot       string
  HttpsCertFile string
  HttpsKeyFile  string
  ClientBKS     string
  ClientBKSPassword string
}

var Server = ServerConfig{
  Port:          4280,
  URLRoot:       "http://localhost:4280/",
  HttpsCertFile: "",
  HttpsKeyFile:  "",
  ClientBKS:     "./keystore.bks",
  ClientBKSPassword: "password",
}

type GCMConfig struct {
  OAuthToken string
}

var GCM = GCMConfig{
  OAuthToken: "",
}

type ExclusionIntervalConfig struct {
  Start      string
  Duration   string
  DaysOfWeek []time.Weekday // int, 0 - 6, 0 = Sunday
}
type SensorConfig struct {
  Mode               string
  MockTTY            bool
  Names              map[string]string
  Types              map[string]types.SensorType
  TTYPath            string
  AjarThreshold      time.Duration
  ExclusionIntervals []ExclusionIntervalConfig
}

var Sensor = SensorConfig{
  Mode:               "TTY",
  MockTTY:            false,
  Names:              make(map[string]string),
  Types:              make(map[string]types.SensorType),
  TTYPath:            "/dev/ttyUSB0",
  AjarThreshold:      30 * time.Second,
  ExclusionIntervals: make([]ExclusionIntervalConfig, 0),
}

type CameraSpecConfig struct {
  Url      string
  Interval int
  Count    int
}
type PhotoConfig struct {
  Retention  string
  Directory  string
  CameraSpec map[string][]CameraSpecConfig
}

var Photo = PhotoConfig{
  Retention:  "720h",
  Directory:  "./photos",
  CameraSpec: make(map[string][]CameraSpecConfig),
}

type UserAuthConfig struct {
  OAuthAudience          string
  OAuthClientID          string
  GoogleOAuthCertsURL    string
  GoogleAccountWhitelist []string
}

var UserAuth = UserAuthConfig{
  OAuthAudience:          "",
  OAuthClientID:          "",
  GoogleOAuthCertsURL:    "https://www.googleapis.com/oauth2/v1/certs",
  GoogleAccountWhitelist: make([]string, 0),
}

type PathType int

const (
  PATH_REGID PathType = iota
  PATH_HEARTBEAT
  PATH_RECENT
  PATH_PHOTO_LIST
  PATH_PHOTO
)

type URLPathConfig struct {
  RegID      string
  Heartbeat  string
  Recent     string
  PhotoList  string
  PhotoFetch string
  QRConfig   string
}

var URLPath = URLPathConfig{
  Heartbeat:  "/heartbeat",
  PhotoFetch: "/photo/",
  PhotoList:  "/photos/",
  QRConfig:   "/qrconfig",
  Recent:     "/recent",
  RegID:      "/regid",
}

func init() {
  // locate a config file
  var configFile string
  flag.StringVar(&configFile, "config", "./config.json", "fully qualified path to the JSON config file")
  flag.Parse()

  // load the contents of that config file
  file, err := os.Open(configFile)
  if err != nil {
    log.Fatal("loading config failed opening the config file '"+configFile+"'", err)
  }
  jsonText, err := ioutil.ReadAll(file)
  if err != nil {
    log.Fatal("loading config failed reading the config file '"+configFile+"'", err)
  }

  // parse the JSON config contents into memory
  type jsonConfig struct {
    General  *GeneralConfig
    Server   *ServerConfig
    GCM      *GCMConfig
    Sensor   *SensorConfig
    Photo    *PhotoConfig
    UserAuth *UserAuthConfig
    URLPath  *URLPathConfig
  }
  // this block assigns the top-level package objects as the destination of
  // the JSON parse operation. Since these instances are populated with
  // defaults above, JSON will overwrite the defaults if and only if present
  // in the file.
  jsonTarget := jsonConfig{
    General:  &General,
    Server:   &Server,
    GCM:      &GCM,
    Sensor:   &Sensor,
    Photo:    &Photo,
    UserAuth: &UserAuth,
    URLPath:  &URLPath,
  }
  err = json.Unmarshal([]byte(jsonText), &jsonTarget)
  if err != nil {
    if serr, ok := err.(*json.SyntaxError); ok {
      lines := strings.Split(string(jsonText), "\n")
      target := int(serr.Offset)
      seen := 0
      for i, line := range lines {
        if target <= (seen + len(line) + 1) { // assume ASCII
          fmt.Println(line)
          log.Fatal("JSON parse error at line " + strconv.Itoa(i+1) + ", column " + strconv.Itoa(target-seen))
        }
        seen += len(line) + 1
      }
    }
    log.Fatal("loading config failed on unmarshal ", err)
  }

  // set up logging based on config
  if General.Debug {
    plog.SetLogLevel(plog.LEVEL_DEBUG)
  }
  if General.LogFile != "" && !General.Debug {
    plog.SetLogFile(General.LogFile)
  }

  common.SensorState = make(map[string]types.Sensor)
  cnt := 0
  for id, name := range Sensor.Names {
    kind, ok := Sensor.Types[id]
    if !ok {
      log.Fatal("missing sensor type spec for '" + id + "'")
    }
    common.SensorState[id] = types.Sensor{Name: name, ID: id, Kind: kind}
    cnt += 1
  }
  if cnt == 0 {
    log.Fatal("no sensor names configured")
  }
}

/* Returns the full URL (using URL/host as configured in config.json) for a
 * particular URL path */
func GetURLFor(path PathType) string {
  pathStr := map[PathType]string{
    PATH_REGID:      URLPath.RegID,
    PATH_HEARTBEAT:  URLPath.Heartbeat,
    PATH_RECENT:     URLPath.Recent,
    PATH_PHOTO_LIST: URLPath.PhotoList,
    PATH_PHOTO:      URLPath.PhotoFetch,
  }[path]
  // make sure we don't end up with duplicate /-es in URLs
  left := strings.TrimRight(Server.URLRoot, "/")
  right := strings.TrimLeft(pathStr, "/")
  return strings.Join([]string{left, right}, "/")
}

/* Joins the indicated strings URL-style, i.e. separated by "/" and such that
 * there is exactly one "/" between joined segments.
 */
func URLJoin(chunks ...string) string {
  nChunks := len(chunks)
  trimmed := make([]string, nChunks)
  trimmed[0] = strings.TrimRight(chunks[0], "/")
  for i, chunk := range chunks[1:nChunks] {
    trimmed[i+1] = strings.TrimRight(strings.TrimLeft(chunk, "/"), "/")
  }
  trimmed[nChunks-1] = strings.TrimLeft(chunks[nChunks-1], "/")
  return strings.Join(trimmed, "/")
}

/* Returns a string encoding all the config settings the Android app client
 * needs, in the standard Java properties file format. */
func GetClientConfig() (string, error) {
  template := `OAUTH_AUDIENCE=audience:server:client_id:%v
REGID_URL=%v
PHOTO_BASE=%v
CANONICAL_SERVER_NAME=%v
KEYSTORE_PASSWORD=%v
KEYSTORE=%v
`
  serverUrl, err := url.Parse(Server.URLRoot)
  if err != nil {
    plog.Warn("config.GetClientConfig", "error parsing URLRoot?!", err)
    return "", err
  }

  canonicalName := serverUrl.Host
  colon := strings.Index(canonicalName, ":")
  if colon == -1 {
    canonicalName = canonicalName + ":-1"
  }

  ksFile, err := os.Open(Server.ClientBKS)
  if err != nil {
    plog.Error("server.keystore", "no client BKS found")
    return "", err
  }
  defer ksFile.Close()

  ksBytes, err := ioutil.ReadAll(ksFile)
  if err != nil {
    plog.Error("server.keystore", "failed reading BKS file "+Server.ClientBKS)
    return "", err
  }

  ksB64 := base64.StdEncoding.EncodeToString(ksBytes)

  populated := fmt.Sprintf(template, UserAuth.OAuthAudience, GetURLFor(PATH_REGID), GetURLFor(PATH_PHOTO_LIST), canonicalName, Server.ClientBKSPassword, ksB64)
  return populated, nil
}

/* Returns a URL pointing to a QR code that encodes the config data as
 * returned by GetClientConfig() */
func GetClientConfigQR() (string, error) {
  configText, err := GetClientConfig()
  if err != nil {
    return "", err
  }
  escapedText := url.QueryEscape(configText)
  return General.QRGenURL + escapedText, nil
}

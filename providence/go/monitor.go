package main
import (
  "bufio"
  "bytes"
  "database/sql"
  "encoding/json"
  "flag"
  "io"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strconv"
  "strings"
  "time"
  _ "github.com/mattn/go-sqlite3"
)

type ajarAction int
const (
  ALARM ajarAction = iota
  RECORD
  NOTIFY
)
/* Global config structure. Should be populated before any goroutines are
 * spawned, because it's not synchronized. Similarly, should be read-only
 * after first populated. */
var config struct {
  Tty string
  HttpPort int
  DatabasePath string
  SensorNames []struct {
    Id string
    Name string
  }
  AjarThreshold int
  AjarRules []struct {
    Threshold int
    Action []ajarAction
  }
  ExclusionIntervals []struct {
    Hour int
    Min int
    Duration int
    DaysOfWeek []time.Weekday // int, 0 - 6, 0 = Sunday
  }
}
func loadConfig() {
  var flagPort int
  var flagTty string
  var flagDatabase string
  var configFile string
  flag.StringVar(&configFile, "config", "./monitor.json", "fully qualified path to the JSON config file")
  flag.IntVar(&flagPort, "port", 4280, "port of the HTTP server")
  flag.StringVar(&flagTty, "tty", "/dev/ttyUSB0", "USB TTY file connected to the Arduino")
  flag.StringVar(&flagDatabase, "database", "./monitor.sqlite3", "fully qualified path to the sqlite3 database file")
  flag.Parse()

  file, err := os.Open(configFile)
  if err != nil {
    log.Fatal("loadConfig failed opening the config file '" + configFile + "'", err)
  }
  jsonText, err := ioutil.ReadAll(file)
  if err != nil {
    log.Fatal("loadConfig failed reading the config file '" + configFile + "'", err)
  }
  err = json.Unmarshal([]byte(jsonText), &config)
  if flagTty != "" {
    config.Tty = flagTty
  }
  if flagPort > 0 {
    config.HttpPort = flagPort
  }
  if flagDatabase != "" {
    config.DatabasePath = flagDatabase
  }
}

/* Constants representing the physical nature of the sensor hardware */
type sensorType int
const (
  window = iota
  door
  motion
)
/* Constants representing the kinds of events that can happen on sensors. Some
 * of these are electrical(trip, reset), others are abstract (door is ajar).
 */
type eventCode int
const (
  TRIP = iota
  RESET
  AJAR
  AJAR_RESOLVED
  ANOMALY
)
/* Represents a monitorable event that has occurred. */
type event struct {
  Which string
  Type sensorType
  Action eventCode
  // TOOD: timestamp?
}

/* Reads a USB TTY looking for JSON messages from a hardware monitor and
 * injects low-level (trip and reset) eventCodes into the outgoing channel.
 * Never reads from 'incoming'; accordingly, should never be registered for
 * any message types or it will eventually deadlock when the channel buffer
 * fills.
 */
func ttyreader(incoming chan event, outgoing chan event) {
  file, err := os.Open(config.Tty)
  if err != nil {
    log.Fatal("Error opening ", config.Tty, ", aborting ", err)
    return
  }
  reader := bufio.NewReader(file)
  dec := json.NewDecoder(reader)
  var event event
  for {
    dec.Decode(&event)
    outgoing <- event
  }
}

/* Logs all events it gets to a sqlite3 database. Should be registered for all
 * eventCodes. Never sends anything to the outgoing channel.
 */
func recorder(incoming chan event, outgoing chan event) {
  db, err := sql.Open("sqlite3", config.DatabasePath)
  if err != nil {
    log.Print("ERROR: recorder failed to open ", config.DatabasePath, err)
  }
  defer db.Close()
  insert, err := db.Prepare("insert into events (name, value) values (?, ?)")
  if err != nil {
    log.Print("ERROR: recorder failed to prepare insert statement ", err)
  }
  defer insert.Close()
  for {
    event := <-incoming
    insert.Exec(event.Which, event.Action)
  }
}

/* Looks for low-level events on the incoming channel and applies some
 * heuristics to determine whether they are noteworthy. Will inject
 * higher-level eventCodes (ajar, anomalous) to the outgoing channel as
 * appropriate.
 *
 * Currently the heuristics are exclusion intervals and 'door is ajar'
 * detection. Should only be registered for low-level events.
 */
func monitor(incoming chan event, outgoing chan event) {
  // local structs used in synthesizing human-meaningful events from raw events
  type timeWindow struct {
    hour int
    minute int
    duration time.Duration
    dows []time.Weekday
  }
  type tripRecord struct {
    when time.Time
    nextSend time.Duration
  }

  // initialize the structs representing exclusion windows
  workdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
  weekends := []time.Weekday{time.Saturday, time.Sunday}
  windows := []timeWindow{
    timeWindow{7, 30, 2 * time.Hour + 30 * time.Minute, workdays[:]},
    timeWindow{17, 30, 4 * time.Hour + 30 * time.Minute, workdays[:]},
    timeWindow{9,  0, 1 * time.Hour, weekends[:]}, // TODO: remove
    //timeWindow{9,  0, 11 * time.Hour, weekends[:]},
  }
  ajarThreshold := 5 * time.Second
  resendFrequency := 1 * time.Minute
  lastTrips := make(map[string]*tripRecord)
  ticker := time.Tick(1 * time.Second)
  for {
    select {
    case e := <-incoming:
      now := time.Now()

      // timestamp trips for ajar-detection, and clear on resets
      if e.Action == TRIP {
        lastTrips[e.Which] = &tripRecord{now, ajarThreshold}
      } else if e.Action == RESET {
        delete(lastTrips, e.Which)
      }

      // check trips against exclusion intervals for anomalous events
      if e.Action == TRIP {
        inWindow := false
        for _, w := range windows {
          legit := false
          for _, dow := range w.dows {
            if now.Weekday() == dow {
              legit = true
              break
            }
          }
          if !legit {
            continue
          }
          start := time.Date(now.Year(), now.Month(), now.Day(), w.hour, w.minute, 0, 0, time.Local)
          end := start.Add(w.duration)
          if now.After(start) && now.Before(end) {
            inWindow = true
            break
          }
        }
        if !inWindow {
          outgoing <- event{e.Which, 0, ANOMALY}
        }
      }

    case <- ticker:
      // once per second, check whether anything is (still) Ajar & (re)transmit if it's time to
      for which, last := range lastTrips {
        if time.Since(last.when) > ajarThreshold && time.Since(last.when) > last.nextSend {
          last.nextSend += resendFrequency
          outgoing <- event{which, 0, AJAR}
        }
      }
    }
  }
}

/* Message object sent to the reg ID persistence sink. */
type regIdUpdate struct {
  RegId string
  CanonicalRegId string
  Remove bool
}
/* Spin up a goroutine that records user devices' change requests to the reg
 * ID list in the SQLite database.
 */
func regIdPersisterEscalatorHelper() chan regIdUpdate {
  updateChan := make(chan regIdUpdate)

  go func (updateChan chan regIdUpdate) {
    db, err := sql.Open("sqlite3", config.DatabasePath)
    if err != nil {
      log.Fatal("ERROR: escalator failed opening ", config.DatabasePath)
      return
    }
    defer db.Close()

    // prepare a bunch of statements
    insertStmt, err := db.Prepare("insert or ignore into reg_ids values (?)")
    if err != nil {
      log.Print("regId updater failed to prepare insert", err)
    }
    defer insertStmt.Close()
    updateStmt, err := db.Prepare("update reg_ids set reg_id=? where reg_id=?")
    if err != nil {
      log.Print("regId updater failed to prepare update", err)
    }
    defer updateStmt.Close()
    deleteStmt, err := db.Prepare("delete from reg_ids where reg_id=?")
    if err != nil {
      log.Print("regId updater failed to prepare delete", err)
    }
    defer deleteStmt.Close()

    // listen for updates & take the appropriate action
    for {
      select {
      case update := <-updateChan:
        if update.Remove {
          // Basic delete.
          _, err = deleteStmt.Exec(update.RegId)
          if err != nil {
            log.Print("WARNING: failed on insert", err)
          }
        } else if update.CanonicalRegId != "" {
          // To handle the case where the server sends us a canonicalization
          // correction for a regID that isn't actually in the database, we
          // first insert the old one (with the statement set to no-op if
          // already present) and then execute the update. We do these in a
          // transaction to avoid race conditions. This case should actually
          // never happen, since if we don't have a given regID, we can't send
          // it to the server so we can't get a correction for it. So this is
          // a lot of work to cover a case that should never happen. But this
          // does defend against the in-memory map getting out of sync with
          // the persistence store.
          tx, err := db.Begin()
          if err != nil {
            log.Print("WARNING: failed to start a transaction", err)
            break
          }
          _, err = tx.Stmt(insertStmt).Exec(update.RegId)
          if err != nil {
            log.Print("WARNING: failed on update insert", err)
            tx.Rollback()
            break
          }
          _, err = tx.Stmt(updateStmt).Exec(update.CanonicalRegId, update.RegId)
          if err != nil {
            log.Print("WARNING: failed on update update", err)
            tx.Rollback()
            break
          }
          err = tx.Commit()
          if err != nil {
            log.Print("WARNING: failed on update commit", err)
            tx.Rollback()
            break
          }
        } else {
          // Basic insert. Note that the table is NOT NULL UNIQUE on reg IDs,
          // so we can get use the "INSERT OR IGNORE" query form.
          _, err = insertStmt.Exec(update.RegId)
          if err != nil {
            log.Print("WARNING: failed on insert", err)
          }
        }
      }
    }
  }(updateChan)

  return updateChan
}

/* Represents a registration ID operation from a user device */
type regIdRequest struct {
  regId string
  remove bool
}
/* Spins up an HTTP server in a goroutine to which user devices make requests
 * to add & delete registration IDs, per the GCM spec. Server also implements
 * a trivial heartbeat URL that devices can use to detect if the monitor goes
 * offline, and notify locally.
 */
func escalatorHttpHelper() chan regIdRequest {
  regIdRequestChan := make(chan regIdRequest, 5)
  go func (regIdRequestChan chan regIdRequest) {
    // registration ID handler; RESTful:
    // - POST = add reg ID(s) listed in body
    // - DELETE = discard reg ID(s) listed in body
    http.HandleFunc("/regid", func(writer http.ResponseWriter, req *http.Request) {
      body, err := ioutil.ReadAll(req.Body)
      if err != nil {
        log.Print("HTTP request read failure", err)
      } else {
        remove := req.Method == "DELETE"
        for _, s := range strings.Split(string(body), "\n") {
          regIdRequestChan <- regIdRequest{s, remove}
        }
      }
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "OK")
    })

    // heartbeat URL: always returns code=200 with message of "HI"
    http.HandleFunc("/heartbeat", func(writer http.ResponseWriter, req *http.Request) {
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "HI")
    })

    // listen on the configured port
    log.Print(http.ListenAndServe(":" + strconv.Itoa(config.HttpPort), nil))
  }(regIdRequestChan)
  return regIdRequestChan
}

/* Returns all GCM registration (device) IDs stored in the database. This DB
 * is used for the persistent store for these IDS, so this function exists to
 * be called on startup. The escalator is expected to keep the resulting
 * stucture up to date, once loaded.
 *
 * Uses a map to get fast lookups. This structure is expected to change only
 * rarely.
 */
func loadKnownRegIds() map[string]int {
  regIds := make(map[string]int)
  db, err := sql.Open("sqlite3", config.DatabasePath)
  if err != nil {
    log.Fatal("ERROR: escalator failed opening ", config.DatabasePath)
    return nil
  }
  defer db.Close()
  rows, err := db.Query("select * from reg_ids")
  if err != nil {
    log.Print("ERROR: escalator failed to unmarshal known regIds")
    return nil
  }
  for rows.Next() {
    var regId string
    rows.Scan(&regId)
    regIds[regId] = 0
  }

  return regIds
}

/* Watches for higher-level event types and escalates them for
 * human review -- i.e. via GCM. Should only be registered for ajar and
 * anomalous.
 */
func gcmEscalator(incoming chan event, outgoing chan event) {
  // start database persister thread, and load stored list from last execution
  regIdUpdateSink := regIdPersisterEscalatorHelper()
  regIds := loadKnownRegIds()
  var regIdList []string // done up front so we don't repeat this work for every GCM message
  for k := range regIds {
    regIdList = append(regIdList, k)
  }

  // start the HTTP server which is our source for regID creates & deletes
  regIdHttpSource := escalatorHttpHelper()

  // check each raw event and synthesize higher level events as appropriate
  for {
    select {
    case regId := <-regIdHttpSource:
      // HTTP server is reporting a regID create, delete, or update operation
      if regId.remove { // deleting an existing regId
        delete(regIds, regId.regId)
      } else { // adding a new RegId
        regIds[regId.regId] = 0
      }
      regIdUpdateSink <- regIdUpdate{regId.regId, "", regId.remove}
      regIdList = []string{}
      for k := range regIds { // rebuild our cache
        regIdList = append(regIdList, k)
      }
    case ev := <-incoming:
      // New monitoring event from the dispatcher.

      if len(regIds) < 1 {
        log.Print("No registered devices, skipping event", ev)
        break
      }

      // define some constants and structs used only for JSON data formatting
      // & communication with GCM
      const (
        GCM_URL = "https://android.googleapis.com/gcm/send"
        GCM_MIMETYPE = "application/json"
        OAUTH_TOKEN = "AIzaSyDGBuD9xLI0weiV0nz7Z9AT76EyzSMXk7Y"
      )
      type gcmRequest struct {
        RegistrationIds []string `json:"registration_ids"`
        Data interface{} `json:"data"`
      }
      type gcmResponseResults struct {
        MessageId string `json:"message_id"`
        RegistrationId string `json:"registration_id"`
        Error string `json:"error"`
      }
      type gcmResponse struct {
        MulticastId uint64 `json:"multicast_id"`
        Success int `json:"success"`
        Failure int `json:"failure"`
        CanonicalIds int `json:"canonical_ids"`
        Results []gcmResponseResults `json:"results"`
      }

      // format up a GCM JSON message for the event
      j, ok := json.Marshal(gcmRequest{regIdList, ev})
      if ok != nil {
        log.Print("JSON failure during encode for GCM", ok)
        break
      }

      // send the event to GCM server via HTTP POST, per spec
      req, err := http.NewRequest("POST", GCM_URL, bytes.NewReader(j))
      if err != nil {
        log.Print("Failed to create GCM HTTP request", err)
        break
      }
      req.Header.Add("Authorization", "key=" + OAUTH_TOKEN)
      req.Header.Add("Content-Type", GCM_MIMETYPE)
      client := &http.Client{}
      resp, err := client.Do(req)
      if err != nil {
        log.Print("GCM request failed during execution", err)
        break
      }
      defer resp.Body.Close()

      // look at the JSON response from GCM server & take any actions indicated
      body, err := ioutil.ReadAll(resp.Body)
      if err == nil && len(body) > 0 {
        log.Print("GCM server response:")
        log.Print("        " + string(body))
        var jsonResponse gcmResponse
        jsonErr := json.Unmarshal(body, &jsonResponse)
        if jsonErr != nil {
          log.Print("JSON unmarshal failure on GCM response", jsonErr)
        }
        for i, oldId := range regIdList {
          result := jsonResponse.Results[i]

          // GCM server sent a "canonical registration ID" message; update our list accordingly
          if result.RegistrationId != "" {
            regIdUpdateSink <- regIdUpdate{oldId, result.RegistrationId, false}
            delete(regIds, oldId)
            regIds[result.RegistrationId] = 0
            regIdList = []string{}
            for k := range regIds {
              regIdList = append(regIdList, k)
            }
          }

          // TODO: inspect result for failure; need to determine a retry policy, first
        }
      } else {
        log.Print("HTTP error or empty response from GCM ", err, " ", string(body))
      }
    }
  }
}


/*
 * Main dispatcher loop
 */

/* Stores handler function and its state and registration info. */
type handler struct {
  f func(chan event, chan event)
  ch chan event
  eventCodes map[eventCode]int
}

func main() {
  loadConfig()

  events := make(chan event, 10)

  // make a struct for each handler function, mapping it to events it wants to know about
  handlers := []handler{
    handler{ttyreader, make(chan event, 10), make(map[eventCode]int)}, // no registrations
    handler{recorder, make(chan event, 10), map[eventCode]int{TRIP: 1, RESET: 1, AJAR: 1, ANOMALY: 1}},
    handler{monitor, make(chan event, 10), map[eventCode]int{TRIP: 1, RESET: 1}},
    handler{gcmEscalator, make(chan event, 10), map[eventCode]int{AJAR: 1, ANOMALY: 1}},
  }
  for _, h := range handlers {
    go h.f(h.ch, events)
  }

  // simply loop forever, sending generated events to the listeners who want to hear them
  for {
    evt := <-events
    log.Print(evt.Which + " ", map[eventCode]string{TRIP: "Tripped", RESET: "Reset", AJAR: "Ajar", ANOMALY: "Anomaly"}[evt.Action])
    for _, h := range handlers {
      _, ok := h.eventCodes[evt.Action]
      if ok {
        h.ch <- evt
      }
    }
  }
}

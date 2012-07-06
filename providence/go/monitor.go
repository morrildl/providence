package main
import (
  "bufio"
  "bytes"
  "database/sql"
  "encoding/json"
  "io"
  "io/ioutil"
  "log"
  "net/http"
  "os"
  "strings"
  "time"
  _ "github.com/mattn/go-sqlite3"
)

type sensorType int
const (
  window = iota
  door
  motion
)

type eventType int
const (
  trip = iota
  reset
  ajar
  anomaly
)

type event struct {
  Which string
  Type sensorType
  Action eventType
}

type eventWindow struct {
  hour int
  minute int
  duration time.Duration
  dows []time.Weekday
}

/* Reads a USB TTY looking for JSON messages from a hardware monitor and
injects low-level (trip and reset) eventTypes into the outgoing channel. Never
reads from 'incoming'; accordingly, should never be registered for any message
types or it will eventually deadlock when the channel buffer fills.
 */
func ttyreader(incoming chan event, outgoing chan event) {
  file, err := os.Open("/dev/ttyUSB0")
  if err != nil {
    log.Fatal("Error opening /dev/ttyUSB0, aborting ", err)
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
 * eventTypes. Never sends anything to the outgoing channel.
 */
func recorder(incoming chan event, outgoing chan event) {
  db, err := sql.Open("sqlite3", "./events.sqlite3")
  if err != nil {
    log.Print("ERROR: recorder failed to open evets.sqlite3 ", err)
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

type tripRecord struct {
  when time.Time
  next_send time.Duration
}

/* Looks for low-level events on the incoming channel and applies some
 * heuristics to determine whether they are noteworthy. Will inject
 * higher-level eventTypes (ajar, anomalous) to the outgoing channel as
 * appropriate.
 *
 * Currently the heuristics are exclusion intervals and 'door is ajar'
 * detection. Should only be registered for low-level events.
 */
func monitor(incoming chan event, outgoing chan event) {
  workdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
  weekends := []time.Weekday{time.Saturday, time.Sunday}
  windows := []eventWindow{
    eventWindow{7, 30, 2 * time.Hour + 30 * time.Minute, workdays[:]},
    eventWindow{17, 30, 4 * time.Hour + 30 * time.Minute, workdays[:]},
    eventWindow{9,  0, 1 * time.Hour, weekends[:]}, // TODO: remove
    //eventWindow{9,  0, 11 * time.Hour, weekends[:]},
  }
  // TODO ajar_threshold := 5 * time.Minute
  // TODO resend_frequency := 10 * time.Minute
  ajar_threshold := 5 * time.Second
  resend_frequency := 1 * time.Minute
  last_trips := make(map[string]*tripRecord)
  ticker := time.Tick(1 * time.Second)
  for {
    select {
    case e := <-incoming:
      now := time.Now()

      // timestamp trips for ajar-detection, and clear on resets
      if e.Action == trip {
        last_trips[e.Which] = &tripRecord{now, ajar_threshold}
      } else if e.Action == reset {
        delete(last_trips, e.Which)
      }

      // check trips against exclusion intervals for anomalous events
      if e.Action == trip {
        in_window := false
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
            in_window = true
            break
          }
        }
        if !in_window {
          outgoing <- event{e.Which, 0, anomaly}
        }
      }

    case <- ticker:
      // once per second, check whether anything is (still) Ajar & (re)transmit if it's time to
      for which, last := range last_trips {
        if time.Since(last.when) > ajar_threshold && time.Since(last.when) > last.next_send {
          last.next_send += resend_frequency
          outgoing <- event{which, 0, ajar}
        }
      }
    }
  }
}

type gcmRequest struct {
  RegistrationIds []string `json:"registration_ids"`
  Data interface{} `json:"data"`
}

const (
  GCM_URL = "https://android.googleapis.com/gcm/send"
  GCM_MIMETYPE = "application/json"
  SENDER_ID = "25235963451"
  OAUTH_TOKEN = "AIzaSyDGBuD9xLI0weiV0nz7Z9AT76EyzSMXk7Y"
)

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

/* Looks for higher-level event types and escalates them for
 * human review. Should only be registered for ajar and anomalous.
 */
func escalator(incoming chan event, outgoing chan event) {
  // populate the map of known regIds
  regIds := make(map[string]int)
  db, err := sql.Open("sqlite3", "./events.sqlite3")
  if err != nil {
    log.Print("ERROR: escalator failed opening events.sqlite3")
    return
  }
  defer db.Close()
  rows, err := db.Query("select * from reg_ids")
  if err != nil {
    log.Print("ERROR: escalator failed to unmarshal known regIds")
    return
  }
  for rows.Next() {
    var regId string
    rows.Scan(&regId)
    regIds[regId] = 0
  }

  // spin off a thread to keep the database up to date
  type regIdUpdate struct {
    RegId string
    CanonicalRegId string
    Remove bool
  }
  regIdUpdateChan := make(chan regIdUpdate)
  go func (updateChan chan regIdUpdate) {
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
    for {
      select {
      case update := <-updateChan:
        if update.Remove {
          _, err = deleteStmt.Exec(update.RegId)
          if err != nil {
            log.Print("WARNING: failed on insert", err)
          }
        } else if update.CanonicalRegId != "" {
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
          _, err = insertStmt.Exec(update.RegId)
          if err != nil {
            log.Print("WARNING: failed on insert", err)
          }
        }
      }
    }
  }(regIdUpdateChan)

  // spin off an HTTP server in a thread to listen for registration IDs & heartbeat
  type regIdType struct {
    regId string
    remove bool
  }
  regListener := make(chan regIdType, 5)
  go func (regChan chan regIdType) {
    http.HandleFunc("/regid", func(writer http.ResponseWriter, req *http.Request) {
      body, err := ioutil.ReadAll(req.Body)
      if err != nil {
        log.Print("HTTP request read failure", err)
      } else {
        remove := req.Method == "DELETE"
        for _, s := range strings.Split(string(body), "\n") {
          regChan <- regIdType{s, remove}
        }
      }
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "OK")
    })
    http.HandleFunc("/heartbeat", func(writer http.ResponseWriter, req *http.Request) {
      writer.WriteHeader(http.StatusOK)
      io.WriteString(writer, "HI")
    })
    log.Print(http.ListenAndServe(":4280", nil))
  }(regListener)

  // the reason we're here: check each raw event and synthesize higher level
  // events as appropriate
  var regIdList []string
  for k := range regIds {
    regIdList = append(regIdList, k)
  }
  for {
    select {
    case regId := <-regListener:
      if regId.remove {
        delete(regIds, regId.regId)
      } else {
        regIds[regId.regId] = 0
      }
      regIdUpdateChan <- regIdUpdate{regId.regId, "", regId.remove}
      regIdList = []string{}
      for k := range regIds {
        regIdList = append(regIdList, k)
      }
    case ev := <-incoming:
      if len(regIds) < 1 {
        log.Print("No registered devices, skipping event", ev)
        break
      }
      j, ok := json.Marshal(gcmRequest{regIdList, ev})
      if ok == nil {
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
            if result.RegistrationId != "" {
              regIdUpdateChan <- regIdUpdate{oldId, result.RegistrationId, false}
              delete(regIds, oldId)
              regIds[result.RegistrationId] = 0
              regIdList = []string{}
              for k := range regIds {
                regIdList = append(regIdList, k)
              }
            }
          }
        } else {
          log.Print("HTTP error or empty response from GCM", err, err == nil, len(body), body, string(body))
        }
      } else {
        log.Print("JSON failure during encode for GCM", ok)
      }
    }
  }
}

/* Stores handler function and its state and registration info. */
type handler struct {
  f func(chan event, chan event)
  ch chan event
  eventTypes map[eventType]int
}

func main() {
  events := make(chan event, 10)

  handlers := []handler{
    handler{ttyreader, make(chan event, 10), make(map[eventType]int)}, // no registrations
    handler{recorder, make(chan event, 10), map[eventType]int{trip: 1, reset: 1, ajar: 1, anomaly: 1}},
    handler{monitor, make(chan event, 10), map[eventType]int{trip: 1, reset: 1}},
    handler{escalator, make(chan event, 10), map[eventType]int{ajar: 1, anomaly: 1}},
  }
  for _, h := range handlers {
    go h.f(h.ch, events)
  }

  for {
    evt := <-events
    log.Print(evt.Which + " ", map[eventType]string{trip: "Tripped", reset: "Reset", ajar: "Ajar", anomaly: "Anomaly"}[evt.Action])
    for _, h := range handlers {
      _, ok := h.eventTypes[evt.Action]
      if ok {
        h.ch <- evt
      }
    }
  }
}

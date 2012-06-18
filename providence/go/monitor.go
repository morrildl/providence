package main
import (
  "bufio"
  "bytes"
  "database/sql"
  "encoding/json"
  "fmt"
  "net/http"
  "io/ioutil"
  "os"
  "time"
  _ "github.com/mattn/go-sqlite3"
)

const (
  ESCALATOR_URL = "http://medea:9000/escalate"
  ESCALATOR_MIMETYPE = "text/json"
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
    fmt.Println("Error opening /dev/ttyUSB0, aborting")
    fmt.Println(err)
    return
  }
  reader := bufio.NewReader(file)
  dec := json.NewDecoder(reader)
  var event event
  for {
    dec.Decode(&event)
    fmt.Println("ttyreader sending", event.Which, event.Action)
    outgoing <- event
  }
}

/* Logs all events it gets to a sqlite3 database. Should be registered for all
 * eventTypes. Never sends anything to the outgoing channel.
 */
func recorder(incoming chan event, outgoing chan event) {
  db, err := sql.Open("sqlite3", "./events.sqlite3")
  if err != nil {
    fmt.Println(err)
  }
  defer db.Close()
  insert, err := db.Prepare("insert into events (name, value) values (?, ?)")
  if err != nil {
    fmt.Println(err)
  }
  defer insert.Close()
  for {
    event := <-incoming
    insert.Exec(event.Which, event.Action)
    if err != nil {
      fmt.Println(err)
    }
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
    eventWindow{5, 30, 4 * time.Hour + 30 * time.Minute, workdays[:]},
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

/* Looks for higher-level event types and escalates them for
 * human review. Should only be registered for ajar and anomalous.
 */
func escalator(incoming chan event, outgoing chan event) {
  for {
    select {
    case e := <-incoming:
      j, ok := json.Marshal(e)
      if ok == nil {
        resp, err := http.Post(ESCALATOR_URL, ESCALATOR_MIMETYPE, bytes.NewReader(j))
        if err != nil {
          fmt.Println("HTTP FAIL", err)
          break
        }
        defer resp.Body.Close()
        body, err := ioutil.ReadAll(resp.Body)
        if err != nil && len(body) > 0 {
          fmt.Println("HTTP response", body)
        } else {
          fmt.Println("HTTP error or empty response", err)
        }
      } else {
        fmt.Println("json FAIL", ok)
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
    fmt.Print(evt.Which + " ")
    fmt.Println(map[eventType]string{trip: "Tripped", reset: "Reset", ajar: "Ajar", anomaly: "Anomaly"}[evt.Action])
    for i, h := range handlers {
      _, ok := h.eventTypes[evt.Action]
      if ok {
        fmt.Println("Sending ", evt.Action, evt.Which, "to", i)
        h.ch <- evt
      }
    }
  }
}

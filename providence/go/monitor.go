package main
import (
  "fmt"
  "bufio"
  "os"
  "database/sql"
  "encoding/json"
  _ "github.com/mattn/go-sqlite3"
  "time"
)

type eventType int
const (
  trip = iota
  reset
)
type event struct {
  Which string
  Action eventType
}

func ttyreader(c chan event) {
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
    c <- event
  }
}

func recorder(events chan event) {
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
    event := <-events
    insert.Exec(event.Which, event.Action)
    if err != nil {
      fmt.Println(err)
    }
  }
}

type window struct {
  hour int
  minute int
  duration time.Duration
  dows []time.Weekday
}
func decidererer(events chan event) {
  /*
   * Things it should do:
   * - Declare a set of time windows
   * - If door activity occurs outside that time window, flag
   * - If activity occurs "a lot" regardless of time window, flag
   * - If door looks like it's open, flag
   */
  workdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
  weekends := []time.Weekday{time.Saturday, time.Sunday}
  grr := func(s string) time.Duration {
    d, _ := time.ParseDuration(s)
    return d
  }
  windows := []window{
    window{7, 30, grr("2h30m"), workdays[:]},
    window{5, 30, grr("4h30m"), workdays[:]},
    window{9,  0, grr("11h0m"), weekends[:]},
    window{0,  0, grr("2h0m"), workdays[:]},
  }
  //state := map[string]int{"FRONT_DOOR": 1, "GARAGE_DOOR": 1, "MOTION": 1}
  for {
    e := <-events
    now := time.Now()
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
        fmt.Println("Door event ", e.Which, "/", e.Action, " falls within window ", w.hour, w.minute)
      }
    }
  }
}

type handler struct {
  f func(chan event)
  ch chan event
}

func main() {
  tty := make(chan event, 10)
  go ttyreader(tty)

  funcs := [](func(chan event)){recorder, decidererer}
  handlers := make([]handler, len(funcs), len(funcs))
  for i, f := range funcs {
    handlers[i] = handler{f, make(chan event, 10)}
    go f(handlers[i].ch)
  }

  for {
    evt := <-tty
    fmt.Print(evt.Which + " ")
    fmt.Println(map[eventType]string{trip: "Tripped", reset: "Reset"}[evt.Action])
    for _, h := range handlers {
      h.ch <- evt
    }
  }
}

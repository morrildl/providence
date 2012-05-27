package main
import (
  "fmt"
  "bufio"
  "os"
  "strings"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
)

type eventType int
const (
  trip = iota
  reset
)
type event struct {
  which string
  action eventType
}

func ttyreader(c chan event) {
  // TODO: error handling
  file, _ := os.Open("/dev/ttyUSB0")
  reader := bufio.NewReader(file)
  for {
    line, _, _ := reader.ReadLine()
    fields := strings.Split(string(line), "=")
    if len(fields) < 2 {
      continue
    }
    msg := event{}
    msg.which = fields[0]
    msg.action = map[string]eventType{"TRIP": trip, "RESET": reset}[fields[1]]
    c <- msg
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
    result, err := insert.Exec(event.which, event.action)
    fmt.Println(result)
    if err != nil {
      fmt.Println(err)
    }
  }
}

func main() {
  tty := make(chan event, 10)
  records := make(chan event, 10)
  go ttyreader(tty)
  go recorder(records)
  for {
    line := <-tty
    fmt.Print(line.which + " ")
    fmt.Println(map[eventType]string{trip: "Tripped", reset: "Reset"}[line.action])
    records <- line
  }
}

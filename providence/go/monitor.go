package main
import (
  "fmt"
  "bufio"
  "os"
)

type Message struct {
  s string
}

func ttyreader(c chan Message) {
  file, _ := os.Open("/dev/ttyUSB0")
  reader := bufio.NewReader(file)
  for {
    line, _, _ := reader.ReadLine()
    msg := Message{string(line)}
    c <- msg
  }
}

func main() {
  c := make(chan Message, 10)
  go ttyreader(c)
  for {
    line := <-c
    fmt.Print(line.s + "\n")
  }
}

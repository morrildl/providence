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
package gpio

import (
  "os"
  "syscall"
  "time"

  "providence/common"
  "providence/log"
)

const (
  DEBOUNCE_BINARY time.Duration = 75
  DEBOUNCE_RINGER = 50
)

const (
  TRIP bool = false
  RESET bool = true
)

func makeGpioMonitor(path string) (chan bool, error) {
  ch := make(chan bool, 10)

  // Set up epoll
  efd, err := syscall.EpollCreate(1)
  if err != nil {
    log.Debug("gpio.makeGpioMonitor", "failed in EpollCreate()")
    return nil, err
  }

  // open the actual GPIO input file; expected to be /sys/class/gpio/gpio${PIN}/value
  file, err := os.OpenFile(path, 0 | syscall.O_RDONLY | syscall.O_NONBLOCK, 0)
  if err != nil {
    log.Debug("gpio.makeGpioMonitor", "failed to open GPIO file")
    return nil, err
  }
  gpiofd := int32(file.Fd()) // we need the raw FD for epoll

  // tell epoll() we want EPOLLPRI and EPOLLERR notifications (we get EPOLLERR automatically anyway)
  event := syscall.EpollEvent{Events: 0 | syscall.EPOLLPRI | syscall.EPOLLERR, Fd: gpiofd, Pad: 0}
  if err = syscall.EpollCtl(efd, syscall.EPOLL_CTL_ADD, int(gpiofd), &event); err != nil {
    log.Debug("gpio.makeGpioMonitor", "failed in EpollCtl()")
    return nil, err
  }

  // struct for epoll() to write fd states
  events := make([]syscall.EpollEvent, 1)

  go func() {
    buf := make([]byte, 32) // we should only ever need 1 byte, though

    for {
      count, err := syscall.EpollWait(efd, events, -1)
      if err == nil && count > 0 {
        /*if events[0].Events & syscall.EPOLLERR != 0 || events[0].Events & syscall.EPOLLHUP != 0  {
          fmt.Println("epoll error")
        } else*/
        if events[0].Events & syscall.EPOLLPRI != 0 {
          // file contents changed. seek back to file start and re-read new contents
          if _, err := file.Seek(0, 0); err != nil {
            log.Error("gpio.rawMonitor", "seek failure on " + path, err)
            continue
          }
          count, err := file.Read(buf)
          if err == nil && count > 0 {
            switch {
            case buf[0] == '0':
              ch <- false
            case buf[0] == '1':
              ch <- true
            default:
              log.Error("gpio.rawMonitor", "unexpected GPIO file character " + string(buf[0]))
            }
          } else {
            log.Error("gpio.rawMonitor", "file read failure on " + path, err)
            continue
          }
        } else {
          log.Error("gpio.rawMonitor", "epoll_wait() woke for " + path + " without EPOLLPRI; events=" + string(events[0].Events))
        }
      }
    }
  }()

  return ch, nil
}

/* A binary sensor is a simple normally-closed switch, like a door or window
 * sensor. As a mechanical switch, it needs to be debounced. We accomplish
 * that by simply delaying the channel send by a debounce interval. */
func startBinaryMonitor(path string, outgoing chan common.Event) error {
  monitor, err := makeGpioMonitor(path)
  if err != nil {
    return err
  }

  timer := time.AfterFunc(0, func() {})
  lastSent := RESET

  for {
    state := <-monitor
    timer.Stop()

    // If the switch makes noise and settles back to the same state it was
    // already in within the debounce timeout, don't send a no-op message.
    // e.g. don't send a "door closed" event while the door was already
    // closed. Technically this means you can sneak through the door during
    // the debounce interval, but we're talking about < 100 milliseconds so
    // you'd have to be rather quick about it.
    if state == lastSent {
      continue
    }

    var action common.EventCode
    if state == RESET {
      action = common.RESET
    } else {
      action = common.TRIP
    }

    // Wait briefly before sending the message. If we are indeed settled, the
    // anon func will send the event message in DEBOUNCE_BINARY milliseconds; if
    // we are not settled, the timer.Stop() call above will abort the prior send,
    // and we'll schedule a new one starting from now.
    timer = time.AfterFunc(DEBOUNCE_BINARY * time.Millisecond, func() {
      lastSent = state
      outgoing <- common.Event{Which: common.Sensors[path], Action: action, When: time.Now()}
    })
  }

  return nil
}

/* A ringing sensor is one which alternates rapidly between TRIP and RESET for
 * the duration of the event it is reporting. This is typical of electronic
 * sensors such as motion detectors. */
func createRingerMonitor(path string, outgoing chan common.Event) error {
  monitor, err := makeGpioMonitor(path)
  if err != nil {
    return err
  }

  logicalState := RESET
  timer := time.AfterFunc(0, func(){})

  for {
    rawState := <-monitor
    timer.Stop()

    // the first moment we see the sensor go to TRIP, we know it's going to be
    // the start of a ringing interval, so update our logical state and send
    // the TRIP event immediately
    if rawState == TRIP && logicalState == RESET {
        logicalState = TRIP
        outgoing <- common.Event{Which: common.Sensors[path], Action: common.TRIP, When: time.Now()}
    }

    // when we see sensor go back to RESET, it could be the end of the ringing
    // period, but we can't know for sure until a little time passes without
    // it going back to RESET. so we schedule the switch back to logical RESET
    // state for a brief duration into the future. If the sensor isn't done
    // ringing and falls back to TRIP, the timer.Stop() above will abort this
    // action, and we'll schedule a new one next it RESETs.
    if rawState == RESET {
      timer = time.AfterFunc(DEBOUNCE_RINGER * time.Millisecond, func() {
        logicalState = RESET
        outgoing <- common.Event{Which: common.Sensors[path], Action: common.RESET, When: time.Now()}
      })
    }
  }

  return nil
}

/* Reads 1/0 values from sensors connected to GPIO pins. Pin config is
 * specified in common.Config: if this module is in use, it assumes the pin
 * IDs are actually path names to a /sys/class/gpio values file.
 * Injects low-level (trip and reset) eventCodes into the outgoing channel.
 * Never reads from 'incoming'; accordingly, should never be registered for
 * any message types or it will eventually deadlock when the channel buffer
 * fills.
 */
func Reader(incoming chan common.Event, outgoing chan common.Event) {
  for path, _ := range common.Config.SensorNames {
    var err error
    if common.Config.SensorTypes[path] == common.MOTION {
      err = createRingerMonitor(path, outgoing)
    } else {
      err = startBinaryMonitor(path, outgoing)
    }
    if err != nil {
      log.Error("gpio.Reader", "error opening ", path, ", skipping ", err)
    }
  }
}

var Handler = common.Handler{Reader, make(chan common.Event, 10), map[common.EventCode]int{}}

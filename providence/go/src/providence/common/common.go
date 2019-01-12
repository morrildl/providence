/* Copyright © 2013 Dan Morrill
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

package common

import (
  "providence/types"
  "sync"
)

type eventState struct {
  event types.Event
  touched time.Time
  locker *EventLock
  waiting []chan response
}

var eventCache map[string]eventState

var queue chan request

type response struct {
  lock *EventLock
  err error
}

type operation int
const (
  op_LOCK operation = iota
  op_COMMIT
  op_RELEASE
  op_CREATE
)

type request struct {
  op operation
  sensorID string
  eventID string
  lock *EventLock
  responseChan chan lockResponse
}

func sendAndWait(req request) (types.Event, error) {
  request.responseChan = make(chan response, 1)
  lockQueue <- req
  resp := <-rc
  return rc.Event, rc.err
}

type EventLock struct {
  event types.Event
}

func CreateAndLockEvent(sensorID string) *EventLock {
  event, _ := sendAndWait(request{ op: op_CREATE, sensorID: sensorID })
  return event
}

func LockEvent(eventID string) (*EventLock, error) {
  return sendAndWait(request{ op: op_LOCK, eventID: eventID })
}

func (el *EventLock) Commit() error {
  _, err := sendAndWait(request{ op_COMMIT, el })
  return err
}

func (el *EventLock) Release() error {
  _, err := sendAndWait(request{ op_RELEASE, el })
  return err
}

func pumpWaitingQueue(state eventState) {
  state.locker = nil

  if len(state.waiting) > 0 {
    next = state.waiting[0]
    state.waiting = state.waiting[1:]
    lock := EventLock{ state.event }
    state.locker = &lock
    next <- response{lock: &lock, err: nil}
  }
}

func queueRunner() {
  gc := time.NewTicker(5 * time.Minute)
  for {
    select {

    case req := <-queue:
      select req.op {

      case op_CREATE:
        event := types.NewEvent(req.sensorID)
        state := eventState{
          event: event,
          touched: time.Now(),
          locker: nil,
          waiting: make([]chan response, 0)
        }
        eventCache[event.EventID] = state

        lock := EventLock{ state.event }
        state.locker = &lock
        req.responseChan <- response{lock: &lock, err: nil}

      case op_LOCK:
        state, ok := eventCache[req.eventID]
        if !ok {
          event, err := db.GetEvent(req.eventID)
          if err != nil {
            log.Warning("common.queuerunner:op_LOCK", "attempted lock of non-existent event '"+req.eventID+"'", err)
            req.responseChan <- response{nil, err}
            break
          }
          state = eventState{
            event: event,
            touched: time.Now(),
            locker: nil,
            waiting: make([]chan response, 0)
          }
          eventCache[req.eventID] = state
        }

        if state.locker != nil {
          append(state.waiting, req.responseChan)
        } else {
          lock := EventLock{ state.event }
          state.locker = &lock
          req.responseChan <- response{lock: &lock, err: nil}
        }

      case op_RELEASE:
        eventID := req.lock.event.EventID
        res := response{}

        state, ok := eventCache[eventID]
        if !ok {
          res.err = errors.New("attempted release of unknown event")
          req.rc <- res
          break
        }

        if state.locker != req.lock {
          res.err = errors.New("attempted release by non-lock-holder")
          req.rc <- res
          break
        }

        req.responseChan <- res // err will be nil == success

        state.touched = time.Now()
        pumpWaitingQueue(state)


      case op_COMMIT:
        eventID := req.lock.event.EventID
        res := response{}

        state, ok := eventCache[eventID]
        if !ok {
          res.err = errors.New("attempted commit of unknown event")
          req.rc <- res
          break
        }

        if state.locker != req.lock {
          res.err = errors.New("attempted commit by non-lock-holder")
          req.rc <- res
          break
        }

        req.responseChan <- res // err will be nil == success

        state.event = req.lock.event
        state.touched = time.Now()
        pumpWaitingQueue(state)

      default:
      }

    case <-gc:
      for k, v := range eventCache {
        if time.Since(v.touched) > 10 * time.Minute {
          if len(v.lockQueue) == 0 {
            delete(eventCache, k)
          }
        }
      }
      log.Debug("common.queueRunner", "event cache size after GC: " + strconv.Itoa(len(eventCache)))
    }
  }
}

func init() {
  go queueRunner()
}

type Handler func(chan type.Event, chan type.Event)

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

package db

import (
  "database/sql"
  "errors"

  "providence/common"
  "providence/config"
  "providence/log"
  "providence/types"

  _ "github.com/mattn/go-sqlite3"
)

var (
  db                 *sql.DB
  storeEvent         *sql.Stmt
  selectEvent        *sql.Stmt
  selectRecentEvents *sql.Stmt
  insertRegId        *sql.Stmt
  updateRegId        *sql.Stmt
  deleteRegId        *sql.Stmt
  selectRegId        *sql.Stmt
)

func init() {
  var err error

  // Get a DB connection.
  db, err = sql.Open("sqlite3", config.General.DatabasePath)
  if err != nil {
    log.Error("db.package_init", "recorder failed to open ", config.General.DatabasePath, err)
    panic("recorder failed to open")
  }

  // First, create the tables
  tx, err := db.Begin()
  if err != nil {
    msg := "error creating database tables"
    log.Error("db.package_init", msg, err)
    panic(msg)
  }
  for _, stmt := range []string{
    `CREATE TABLE IF NOT EXISTS Events (
        EventID text not null unique primary key,
        SensorID text not null,
        Trip datetime not null,
        Reset datetime,
        IsAjar integer not null default false,
        IsAnomalous integer not null default false,
        Timestamp datetime not null default(datetime('now')));`,
    `CREATE TABLE IF NOT EXISTS RegIDs (
        RegID text not null unique primary key,
        Timestamp datetime not null default(datetime('now')));`,
  } {
    _, err = tx.Exec(stmt)
    if err != nil {
      msg := "error executing table create statement"
      log.Error("db.package_init", msg, err)
      panic(msg)
    }
  }
  tx.Commit()


  // Initialize events table prepared statements
  storeEvent, err = db.Prepare(
    `insert or replace into events
       (EventID, SensorID, Trip, Reset, IsAjar, IsAnomalous)
     values (?, ?, ?, ?, ?, ?)`)
  if err != nil {
    log.Error("db.package_init", "recorder failed to prepare storeEvent", err)
  }
  selectRecentEvents, err = db.Prepare(
    `select EventID, SensorID, Trip, Reset, IsAjar, IsAnomalous from events
     order by timestamp desc limit 10`)
  if err != nil {
    log.Error("db.package_init", "recorder failed to prepare selectRecentEvents", err)
  }
  selectEvent, err = db.Prepare(`select EventID, SensorID, Trip, Reset, IsAjar, IsAnomalous from events where EventID=?`)
  if err != nil {
    log.Error("db.package_init", "recorder failed to prepare selectEvent", err)
  }


  // Initialize RegIDs table prepared statements
  insertRegId, err = db.Prepare("insert or ignore into RegIDs values (?)")
  if err != nil {
    log.Error("db.package_init", "regId updater failed to prepare insert", err)
  }
  updateRegId, err = db.Prepare("update RegIDs set RegID=? where RegID=?")
  if err != nil {
    log.Error("db.package_init", "regId updater failed to prepare update", err)
  }
  deleteRegId, err = db.Prepare("delete from RegIDs where RegID=?")
  if err != nil {
    log.Error("db.package_init", "regId updater failed to prepare delete", err)
  }
  selectRegId, err = db.Prepare("select RegID from RegIDs")
  if err != nil {
    log.Error("db.package_init", "regId updater failed to prepare select", err)
  }

  // No defer foo.Close() here since this is package init(); when these go out
  // of scope, it will be because process is shutting down
}

/* Logs all events it gets to a sqlite3 database. Should be registered for all
 * eventCodes. Never sends anything to the outgoing channel.
 */
func Recorder(incoming chan types.Event, outgoing chan types.Event) {
  for {
    StoreEvent(<-incoming) // ignore error response since it's already logged
  }
}

/* Message object sent to the RegID persistence sink. */
type RegIdUpdate struct {
  RegId          string
  CanonicalRegId string
  Remove         bool
}

/* Spin up a goroutine that records user devices' change requests to the reg
 * ID list in the SQLite database.
 */
func StartRegIdUpdater() chan RegIdUpdate {
  updateChan := make(chan RegIdUpdate)

  go func() {
    // listen for updates & take the appropriate action
    for {
      select {
      case update := <-updateChan:
        log.Debug("db.regid_updater", "starting persistence request")
        log.Debug("db.regid_updater", update)
        if update.Remove {
          // Basic delete.
          _, err := deleteRegId.Exec(update.RegId)
          if err != nil {
            log.Warn("db.regid_updater", "failed deleting RegID ", err)
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
            log.Warn("db.regid_updater", "failed to start a transaction for canon-RegID update ", err)
            break
          }
          _, err = tx.Stmt(insertRegId).Exec(update.RegId)
          if err != nil {
            log.Warn("db.regid_updater", "failed to insertOrIgnore canon-RegID ", err)
            tx.Rollback()
            break
          }
          _, err = tx.Stmt(updateRegId).Exec(update.CanonicalRegId, update.RegId)
          if err != nil {
            log.Warn("db.regid_updater", "failed on update canon-RegID ", err)
            tx.Rollback()
            break
          }
          err = tx.Commit()
          if err != nil {
            log.Warn("db.regid_updater", "failed to commit canon-RegID transaction", err)
            tx.Rollback()
            break
          }
        } else {
          // Basic insert. Note that the table is NOT NULL UNIQUE on reg IDs,
          // so we can use the "INSERT OR IGNORE" query form.
          _, err := insertRegId.Exec(update.RegId)
          if err != nil {
            log.Warn("db.regid_updater", "failed on RegID insert ", err)
          }
        }
      }
    }
  }()

  return updateChan
}

/* Returns a list of all currently known GCM RegIDs */
func GetRegIds(skip []string) (regIds []string, err error) {
  // Normally I would do the select with a 'where reg_id not in (?)' clause,
  // but there doesn't seem to be a Go Sqlite3 driver that supports slices as
  // args at this time. So instead we filter results manually.

  rowIds := make([]string, 0)
  rows, err := selectRegId.Query()
  if err != nil {
    log.Error("db.get_regids", "failed to fetch known regIds during query ", err)
    return rowIds, err
  } else {
    defer rows.Close()
    for rows.Next() {
      var s string
      rows.Scan(&s)
      add := true
      for _, sk := range skip {
        if s == sk {
          add = false
        }
        break
      }
      if add {
        rowIds = append(rowIds, s)
      }
    }
  }
  return rowIds, nil
}

func GetRecentEvents() ([]types.Event, error) {
  rows, err := selectRecentEvents.Query()
  if err != nil {
    log.Error("db.get_recents", "failed to fetch recent rows ", err)
    return make([]types.Event, 0), err
  }
  defer rows.Close()

  count := 0
  events := make([]types.Event, 10)
  for rows.Next() {
    ev := types.Event{}
    var eventID, sensorID string
    var trip, reset time.Time
    var isAjar, isAnomalous bool
    rows.Scan(&eventID, &sensorID, &trip, &reset, &isAjar, &isAnomalous)
    event := types.Event{eventID, sensorID, trip, reset, isAjar, isAnomalous}
    events[count] = event
    count += 1
    if count == 10 {
      break
    }
  }

  return events[:count], nil
}

func GetEvent(eventID string) (types.Event, error) {
  rows, err := selectEvent.Query(eventID)
  if err != nil {
    log.Error("db.get_recents", "failed to fetch recent rows ", err)
    return types.Event{}, err
  }
  defer rows.Close()

  if !rows.Next() {
    s := "no result found loading event for '" + eventID + "'"
    log.Error("db.GetEvent", s)
    return types.Event{}, errors.New(s)
  }

  var eventID, sensorID string
  var trip, reset time.Time
  var isAjar, isAnomalous bool
  rows.Scan(&eventID, &sensorID, &trip, &reset, &isAjar, &isAnomalous)

  return types.Event{eventID, sensorID, trip, reset, isAjar, isAnomalous}, nil
}

func StoreEvent(event types.Event) error {
  res, err := storeEvent.Exec(event.EventID, event.SensorID, event.Trip, event.Reset, event.IsAjar, event.IsAnomalous)
  if err != nil {
    log.Error("db.StoreEvent", "failed inserting or updating event '" + event.EventID + "'", err)
    return err
  }
  numRows, err := res.RowsAffected()
  if numRows > 1 {
    log.Warning("db.StoreEvent", "multiple rows affected by store operation for '" + event.EventID + "'")
  }
  if numRows < 1 {
    log.Warning("db.StoreEvent", "success but 0 rows affected by store operation for '" + event.EventID + "'")
  }
  return nil
}

var Handler common.Handler = Recorder

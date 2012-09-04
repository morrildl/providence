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
  "log"

  "providence/common"

  _ "github.com/mattn/go-sqlite3"
)

var (
  db *sql.DB
  insertEvent *sql.Stmt
  insertRegId *sql.Stmt
  updateRegId *sql.Stmt
  deleteRegId *sql.Stmt
  selectRegId *sql.Stmt
)

func init() {
  var err error

  // Get a DB connection.
  db, err = sql.Open("sqlite3", common.Config.DatabasePath)
  if err != nil {
    log.Print("ERROR: recorder failed to open ", common.Config.DatabasePath, err)
  }

  // Initialize events table prepared statements
  insertEvent, err = db.Prepare("insert into events (name, value) values (?, ?)")
  if err != nil {
    log.Print("ERROR: recorder failed to prepare insert statement ", err)
  }

  // Initialize reg_ids table prepared statements
  insertRegId, err = db.Prepare("insert or ignore into reg_ids values (?)")
  if err != nil {
    log.Print("regId updater failed to prepare insert", err)
  }
  updateRegId, err = db.Prepare("update reg_ids set reg_id=? where reg_id=?")
  if err != nil {
    log.Print("regId updater failed to prepare update", err)
  }
  deleteRegId, err = db.Prepare("delete from reg_ids where reg_id=?")
  if err != nil {
    log.Print("regId updater failed to prepare delete", err)
  }
  selectRegId, err = db.Prepare("select reg_id from reg_ids")
  if err != nil {
    log.Print("regId updater failed to prepare select", err)
  }

  // No defer foo.Close() here since this is package init(); when these go out
  // of scope, it will be because process is shutting down
}

/* Logs all events it gets to a sqlite3 database. Should be registered for all
 * eventCodes. Never sends anything to the outgoing channel.
 */
func Recorder(incoming chan common.Event, outgoing chan common.Event) {
  for {
    event := <-incoming
    insertEvent.Exec(event.Which.Name, event.Action)
  }
}

/* Message object sent to the RegID persistence sink. */
type RegIdUpdate struct {
  RegId string
  CanonicalRegId string
  Remove bool
}
/* Spin up a goroutine that records user devices' change requests to the reg
 * ID list in the SQLite database.
 */
func StartRegIdUpdater() chan RegIdUpdate {
  updateChan := make(chan RegIdUpdate)

  go func () {
    // listen for updates & take the appropriate action
    for {
      select {
      case update := <-updateChan:
        if common.Config.Debug {
          log.Println("Starting persistence request")
        }
        if update.Remove {
          // Basic delete.
          _, err := deleteRegId.Exec(update.RegId)
          if err != nil {
            log.Println("WARNING: failed deleting RegID ", err)
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
            log.Println("WARNING: failed to start a transaction for canon-RegID update ", err)
            break
          }
          _, err = tx.Stmt(insertRegId).Exec(update.RegId)
          if err != nil {
            log.Println("WARNING: failed to insertOrIgnore canon-RegID ", err)
            tx.Rollback()
            break
          }
          _, err = tx.Stmt(updateRegId).Exec(update.CanonicalRegId, update.RegId)
          if err != nil {
            log.Println("WARNING: failed on update canon-RegID ", err)
            tx.Rollback()
            break
          }
          err = tx.Commit()
          if err != nil {
            log.Println("WARNING: failed to commit canon-RegID transaction", err)
            tx.Rollback()
            break
          }
        } else {
          // Basic insert. Note that the table is NOT NULL UNIQUE on reg IDs,
          // so we can use the "INSERT OR IGNORE" query form.
          _, err := insertRegId.Exec(update.RegId)
          if err != nil {
            log.Println("WARNING: failed on RegID insert ", err)
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
    log.Println("ERROR: failed to fetch known regIds during query ", err)
    return rowIds, err
  } else {
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

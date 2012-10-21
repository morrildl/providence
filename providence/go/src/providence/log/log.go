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

package log

import (
  "fmt"
  "log"
)

type LogLevel int
const (
  LEVEL_ERROR LogLevel = iota
  LEVEL_WARNING
  LEVEL_STATUS
  LEVEL_DEBUG
)

var currentLevel LogLevel = LEVEL_STATUS
func SetLogLevel(newLevel LogLevel) {
  _, ok := levelMap[newLevel]
  if !ok {
    Warn("Logger", "someone tried to set invalid log level ", newLevel)
    return
  }
  currentLevel = newLevel
}

var levelMap map[LogLevel]string = map[LogLevel]string{
  LEVEL_ERROR: "ERROR",
  LEVEL_WARNING: "WARNING",
  LEVEL_STATUS: "STATUS",
  LEVEL_DEBUG: "DEBUG",
}

func doLog(level LogLevel, component string, extras ...interface{}) {
  if level > currentLevel {
    return
  }

  levelString, ok := levelMap[level]
  if !ok {
    levelString = "ERROR"
    Warn("Logger", "called with invalid level ", level)
  }
  var message string
  if _, ok := extras[0].(string); ok {
    message = fmt.Sprintf("[%s] (%s) %s ", levelString, component, extras[0])
    extras = extras[1:]
  } else {
    message = fmt.Sprintf("[%s] (%s) ", levelString, component)
  }
  if len(extras) > 0 {
    extras = append([]interface{}{message}, extras)
  } else {
    extras = []interface{}{message}
  }
  log.Print(extras...)
}

func Debug(component string, extras ...interface{}) {
  doLog(LEVEL_DEBUG, component, extras...)
}

func Error(component string, extras ...interface{}) {
  doLog(LEVEL_ERROR, component, extras...)
}

func Warn(component string, extras ...interface{}) {
  doLog(LEVEL_WARNING, component, extras...)
}

func Status(component string, extras ...interface{}) {
  doLog(LEVEL_STATUS, component, extras...)
}

/*
 * Copyright 2012 Dan Morrill
 * 
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 * 
 *      http://www.apache.org/licenses/LICENSE-2.0
 * 
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License
 */

#define STATE_DUMP_INTERVAL 10000
#define DEBOUNCE_MILLIS 75

// FSM state definitions
#define STATE_START 0
#define STATE_TIME_UNSET 1

// opcode definitions
#define COMMAND_DELIMITER 0xff
#define COMMAND_RESET 0x2a
#define COMMAND_TIME_INIT (COMMAND_RESET + 1)

// state variables
int state = STATE_START;
unsigned long current_millis = 0, state_dump_time = 0;
uint8_t DIGITAL_PIN_STATE[9];
uint64_t DIGITAL_DEBOUNCE[9];
uint16_t ANALOG_PIN_STATE[6];
char* DIGITAL_PIN_NAMES[] = {
  "",
  "",
  "FRONT_DOOR",
  "GARAGE_DOOR",
  "MOTION",
  "",
  "",
  "",
  "",
};

struct wall_time_struct {
  unsigned long raw;
  int init;
  unsigned int hours;
  unsigned int minutes;
  unsigned int seconds;
};
struct wall_time_struct wall_time;

void setup() {
  for (int i = 2; i < 9; ++i) {
    pinMode(i, INPUT);
    digitalWrite(i, LOW);
    DIGITAL_PIN_STATE[i] = 0;
    DIGITAL_DEBOUNCE[i] = 0;
  }
  for (int i = 0; i < 6; ++i) {
    pinMode(i, INPUT);
    ANALOG_PIN_STATE[i] = 0;
  }
  Serial.begin(9600);
}

// clears all state value and drops back to the start state
void reset() {
  state = STATE_TIME_UNSET;
}

// examines the serial/TTL input, looking for command codes, and executes any it finds
void process_incoming() {
  unsigned char cmd_type, opcode = 0;
  unsigned long l = 0, start = 0;
  while (Serial.available() >= 2) { // keep going as long as it we might have messages
    cmd_type = (unsigned char)(Serial.read() & 0xff);
    opcode = (unsigned char)(Serial.read() & 0xff);
    if (cmd_type != COMMAND_DELIMITER) {
      /* if we got gibberish or data was dropped, the delimiter is not the first byte seen,
       * which will cause us to get into a flush loop. This is fine only b/c we don't expect
       * continuous data over the serial port, so it's fine to keep flushing it until the other
       * side pauses in sending. At that point we'll catch up and re-sync with the other side
       */
      Serial.flush();
      return;
    }

    // correctly synced w/ other side on a delimiter byte, now check opcode
    switch (opcode) {
      case COMMAND_RESET:
        state = STATE_START; // eventually will call reset() on next looper pass
        break;
      case COMMAND_TIME_INIT:
        start = millis();
        while ((millis() - start) < 10) {
          if (Serial.available() >= 4)
            break;
        }
        // data shouldn't be arriving slowly or in chunks, so give up after waiting briefly
        if (Serial.available() < 4) {
          Serial.flush(); // results in a flush loop until we catch up with other side
        } else {
          // we have everything we need, now just set the time
          wall_time.raw  = ((((unsigned long)Serial.read()) << 24) & 0xff000000);
          wall_time.raw |= ((((unsigned long)Serial.read()) << 16) & 0xff0000);
          wall_time.raw |= ((((unsigned long)Serial.read()) << 8) & 0xff00);
          wall_time.raw |= (((unsigned long)Serial.read()) & 0xff);
	  wall_time.init = 1;
          //if (state == STATE_TIME_UNSET)
          //  state = STATE_DAYTIME;
        }
        break;
      default:
        // unknown opcode == another flush/resync
        Serial.flush();
    }
  }
}

void update_state() {
  static uint8_t reading;
  static uint64_t current_millis;
  for (int i = 2; i < 9; ++i) {
    if (strcmp(DIGITAL_PIN_NAMES[i], "") == 0) {
      continue;
    }
    reading = digitalRead(i);
    current_millis = millis();
    if (reading != DIGITAL_PIN_STATE[i]) {
      DIGITAL_DEBOUNCE[i] = current_millis;
      DIGITAL_PIN_STATE[i] = reading;
    } else {
      if (DIGITAL_DEBOUNCE[i] != 0) {
        if ((current_millis - DIGITAL_DEBOUNCE[i]) > DEBOUNCE_MILLIS) {
          DIGITAL_DEBOUNCE[i] = 0;
          Serial.print("{\"Which\":\"");
          Serial.print(DIGITAL_PIN_NAMES[i]);
          Serial.print("\",\"Action\":");
          Serial.print(digitalRead(i));
          Serial.println("}");
        }
      }
    }
  }
}

/* Dumps the full state of the system for the other side to peruse. Because we dump our state
 * periodically, we don't need to worry about responding to commands -- the other side can
 * just monitor for changes in state.
 */
void dump_state() {
  Serial.print("state=");
  switch(state) {
    case STATE_START:
      Serial.println("START");
      break;
    case STATE_TIME_UNSET:
      Serial.println("TIME_UNSET");
      break;
  }
  for (int i = 2; i < 9; ++i) {
    if (strcmp(DIGITAL_PIN_NAMES[i], "") == 0) {
      continue;
    }
    Serial.print("{\"Name\":\"");
    Serial.print(DIGITAL_PIN_NAMES[i]);
    Serial.print("\",\"Action\":\"");
    Serial.print(digitalRead(i) == LOW ? "TRIP" : "RESET");
    Serial.println("\"}");
  }
}

void loop() {
  unsigned long tmp = 0;
  static unsigned long prev_millis = 0;

  current_millis = millis();

  if (wall_time.init) {
    tmp = current_millis / 1000;
    if (tmp != prev_millis) {
      prev_millis = tmp;
      wall_time.raw++;
    }
    wall_time.seconds = wall_time.raw % 60;
    wall_time.minutes = (wall_time.raw / 60) % 60;
    wall_time.hours = (wall_time.raw / (60 * 60)) % 24;
  }

  switch(state) {
    case STATE_START:
      reset();
      break;
    case STATE_TIME_UNSET:
      // no-op: do nothing until we get an incoming command to set the time
      break;
  }
  
  process_incoming();

  update_state();
  if ((current_millis - state_dump_time) > STATE_DUMP_INTERVAL) {
    //dump_state();
    state_dump_time = current_millis;
  }
}

extern "C" void __cxa_pure_virtual(void) {
  while(1);
}

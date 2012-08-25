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

// IDs of each pin to be emitted in the output
#define ID_UNUSED 0
#define ID_FRONT_DOOR 1
#define ID_GARAGE_DOOR 2
#define ID_FOYER_MOTION 3
uint8_t PIN_ID[] = { // up to 255 sensors
  ID_UNUSED, // Arduino pins start at 2
  ID_UNUSED, // Arduino pins start at 2

  ID_FRONT_DOOR,
  ID_GARAGE_DOOR,
  ID_FOYER_MOTION,

  ID_UNUSED,
  ID_UNUSED,
  ID_UNUSED,
  ID_UNUSED,
};

// Indicators for how to handle pins, by type of thing connected to them
#define PIN_TYPE_UNUSED 0
#define PIN_TYPE_SWITCH 1 // mechanical switch; needs debouncing
#define PIN_TYPE_RINGER 2 // "rings" 0->1->0 for duration of trip - e.g. motion sensor; cannot be debounced
uint8_t PIN_TYPE[] = {
  PIN_TYPE_UNUSED, // Arduino pins start at 2
  PIN_TYPE_UNUSED, // Arduino pins start at 2

  PIN_TYPE_SWITCH,
  PIN_TYPE_SWITCH,
  PIN_TYPE_RINGER,

  PIN_TYPE_UNUSED,
  PIN_TYPE_UNUSED,
  PIN_TYPE_UNUSED,
  PIN_TYPE_UNUSED,
};

// Tracks the last observed state of each pin
uint8_t PIN_STATE[] = {
  0, 0, 0, 0, 0, 0, 0, 0, 0,
};

// Tracks the time observed value last changed for each pin
uint64_t DEBOUNCE_LAST_CHANGED[] = {
  0, 0, 0, 0, 0, 0, 0, 0, 0,
};

// Per-sensor debounce timeouts. Since electronic signals settle faster than
// analog ones, one timeout does not fit all.
uint8_t DEBOUNCE_TIMEOUT[] = {
  0, // Arduino pins start at 2
  0, // Arduino pins start at 2

  75, // door sensor -- analog; 75ms debounce
  75, // door sensor -- analog; 75ms debounce
  50, // motion sensor -- electronic; 10ms debounce

  0, // unused
  0, // unused
  0, // unused
  0, // unused
};

void setup() {
  for (int i = 2; i < 9; ++i) {
    if (PIN_TYPE[i] != PIN_TYPE_UNUSED) {
      pinMode(i, INPUT);
      digitalWrite(i, LOW); // disable built-in pull-up resistor
    }
  }
  Serial.begin(9600);
}

void loop() {
  static uint8_t reading;
  static uint64_t current_millis;
  for (int i = 2; i < 9; ++i) {
    if (PIN_TYPE[i] == PIN_TYPE_UNUSED) {
      continue;
    }

    // query the world
    reading = digitalRead(i);
    current_millis = millis();

    switch(PIN_TYPE[i]) {
    case PIN_TYPE_SWITCH: // simple debounce of a mechanical switch
      if (reading != PIN_STATE[i]) {
        DEBOUNCE_LAST_CHANGED[i] = current_millis;
        PIN_STATE[i] = reading;
      } 

      // if a debounce is pending, check to see if settle timeout has elapsed
      if (DEBOUNCE_LAST_CHANGED[i] != 0) {
        if ((current_millis - DEBOUNCE_LAST_CHANGED[i]) >= DEBOUNCE_TIMEOUT[i]) {
          DEBOUNCE_LAST_CHANGED[i] = 0;
          Serial.print("{\"Which\":\"");
          Serial.print(PIN_ID[i]);
          Serial.print("\",\"Action\":");
          Serial.print(reading);
          Serial.println("}");
        }
      }
      break;

    case PIN_TYPE_RINGER:
      // A ringer type pins oscillates from 0 to 1 and back, instead of flatly
      // going from 0 to 1. That is, the duration of an event on an analog pin
      // is simply the duration in which the pin reported a value of TRIP; but
      // the duration of an event on a ringer pin is the duration in which the
      // device is ringing its signal line. As a result, a ringer type pin's
      // state is not directly tied to its value. Instead, it enters the
      // TRIP state when the first TRIP reading is seen, but it remains in
      // that state until the value settles back at RESET. So basically we
      // enter TRIP right away, and report a single RESET when we debounce
      // the raw pin reading back to RESET.

      // When we see a transition from RESET to TRIP, record TRIP as the new
      // state of the pin, and immediately report the TRIP event
      if ((reading == 0) && (PIN_STATE[i] == 1)) {
          PIN_STATE[i] = 0;
          Serial.print("{\"Which\":");
          Serial.print(PIN_ID[i]);
          Serial.println(",\"Action\": 0}");
      }
      if (PIN_STATE[i] == 0) {
        if (reading == 0) {
          // If we see a raw TRIP reading on the pin, that means it's not done
          // ringing yet, so update our debounce origin to the current time
          DEBOUNCE_LAST_CHANGED[i] = current_millis;
        } else {
          // Line is reporting RESET, but debounce that signal to make sure
          // it's not still ringing
          if (current_millis - DEBOUNCE_LAST_CHANGED[i] > DEBOUNCE_TIMEOUT[i]) {
            // Ringing is over; report a RESET and take us out of the TRIP state
            Serial.print("{\"Which\":");
            Serial.print(PIN_ID[i]);
            Serial.println(",\"Action\": 1}");
            PIN_STATE[i] = 1;
            DEBOUNCE_LAST_CHANGED[i] = 0;
          }
        }
      }
      // If we are not in a state of TRIP, do nothing
      break;

    default:
      break;
    }
  }
}

extern "C" void __cxa_pure_virtual(void) {
  while(1);
}

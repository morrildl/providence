# Copyright 2012 Dan Morrill
# 
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# 
#      http://www.apache.org/licenses/LICENSE-2.0
# 
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import serial
import time
import sys
import argparse
import threading

ARGS = None

def setup():
  parser = argparse.ArgumentParser(description="Home Sensor Monitor")
  parser.add_argument('-t', '--tty', default='/dev/ttyUSB0', help="the TTY to read from")
  global ARGS
  ARGS = parser.parse_args()

class TtyThread(object):
  def __init__(self):
    pass

if __name__ == '__main__':
  setup()
  print ARGS.tty

def foo():
  if len(sys.argv) != 2:
      print "Please specify a port, e.g. %s /dev/ttyUSB0" % sys.argv[0]
      sys.exit(-1)

  ser = serial.Serial(sys.argv[1])
  ser.setDTR(1)
  time.sleep(0.25)
  ser.setDTR(0)
  ser.close()

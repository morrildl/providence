#!/bin/bash

exec 2> /dev/null

PINS="24 25 27"
GPIO_FILES="active_low direction edge uevent value"
GPIO_PIN_MODE=650
GPIO_FILE_MODE=640
GPIO_GROUP="morrildl"

function error {
  logger -it $(basename $0) "${1:-'Unknown error'}"
  exit 42
}

for PIN in ${PINS}; do
  echo "${PIN}" > /sys/class/gpio/export || error "PIN"
  PIN_DIR="/sys/class/gpio/gpio${PIN}" || error "PIN_DIR"
  echo "in" > ${PIN_DIR}/direction || error "direction"
  echo "both" > ${PIN_DIR}/edge || error "edge"
  echo "0" > ${PIN_DIR}/active_low || error "active_low"

  chgrp ${GPIO_GROUP} ${PIN_DIR} || error "chgrp dir"
  chmod ${GPIO_PIN_MODE} ${PIN_DIR} || error "chmod dir"
  for GPIO_FILE in ${GPIO_FILES}; do
    chgrp ${GPIO_GROUP} ${PIN_DIR}/${GPIO_FILE} || error "chgrp file"
    chmod ${GPIO_FILE_MODE} ${PIN_DIR}/${GPIO_FILE} || error "chmod file"
  done
done

TARGET = $(notdir $(CURDIR))
#INSTALL_DIR = $(HOME)/arduino-0018
INSTALL_DIR=/usr/share/arduino
PORT = /dev/ttyUSB*
UPLOAD_RATE = 57600
AVRDUDE_PROGRAMMER = stk500v1
#MCU = atmega168
MCU = atmega328p
F_CPU = 16000000
OUT_DIR = out
VARIANT = standard

############################################################################
# Below here nothing should be changed...

VERSION=1.0
ARDUINO = $(INSTALL_DIR)/hardware/arduino/cores/arduino
VARIANTS = $(INSTALL_DIR)/hardware/arduino/variants/$(VARIANT)
ARDUINO_LIBS = $(INSTALL_DIR)/libraries
#AVR_TOOLS_PATH = $(INSTALL_DIR)/hardware/tools/avr/bin
AVR_TOOLS_PATH = /usr/bin
AVRDUDE_PATH = $(INSTALL_DIR)/hardware/tools
C_MODULES =  \
$(ARDUINO)/wiring_pulse.c \
$(ARDUINO)/wiring_analog.c \
$(ARDUINO)/wiring.c \
$(ARDUINO)/wiring_digital.c \
$(ARDUINO)/WInterrupts.c \
$(ARDUINO)/wiring_shift.c \
# end of C_MODULES
ARDUINO_MODULES = \
$(OUT_DIR)/wiring_pulse.o \
$(OUT_DIR)/wiring_analog.o \
$(OUT_DIR)/wiring.o \
$(OUT_DIR)/wiring_digital.o \
$(OUT_DIR)/WInterrupts.o \
$(OUT_DIR)/wiring_shift.o


CXX_MODULES = \
$(ARDUINO)/CDC.cpp \
$(ARDUINO)/HID.cpp \
$(ARDUINO)/IPAddress.cpp \
$(ARDUINO)/new.cpp \
$(ARDUINO)/Stream.cpp \
$(ARDUINO)/USBCore.cpp \
$(ARDUINO)/WMath.cpp \
$(ARDUINO)/WString.cpp \
$(ARDUINO)/main.cpp \
$(ARDUINO)/Print.cpp \
$(ARDUINO)/HardwareSerial.cpp \
$(ARDUINO_LIB)/EEPROM/EEPROM.cpp

ARDUINO_CPP_MODULES = \
$(OUT_DIR)/CDC.o \
$(OUT_DIR)/HID.o \
$(OUT_DIR)/IPAddress.o \
$(OUT_DIR)/new.o \
$(OUT_DIR)/Stream.o \
$(OUT_DIR)/USBCore.o \
$(OUT_DIR)/WMath.o \
$(OUT_DIR)/WString.o \
$(OUT_DIR)/main.o \
$(OUT_DIR)/Print.o \
$(OUT_DIR)/HardwareSerial.o

ARDUINO_CPP_LIBS = \
$(OUT_DIR)/EEPROM/EEPROM.o

CXX_APP = $(OUT_DIR)/$(TARGET).cpp
MODULES = $(C_MODULES) $(CXX_MODULES)
SRC = $(C_MODULES)
CXXSRC = $(CXX_MODULES) $(CXX_APP)
FORMAT = ihex


# Name of this Makefile (used for "make depend").
MAKEFILE = Makefile

# Debugging format.
# Native formats for AVR-GCC's -g are stabs [default], or dwarf-2.
# AVR (extended) COFF requires stabs, plus an avr-objcopy run.
#DEBUG = stabs
DEBUG =

OPT = s

# Place -D or -U options here
CDEFS = -DF_CPU=$(F_CPU)L -DARDUINO=$(VERSION)
CXXDEFS = -DF_CPU=$(F_CPU)L -DARDUINO=$(VERSION)

# Place -I options here
CINCS = -I$(ARDUINO) -I$(VARIANTS) -I$(ARDUINO_LIBS)
CXXINCS = -I$(ARDUINO) -I$(VARIANTS) -I$(ARDUINO_LIBS)

# Compiler flag to set the C Standard level.
# c89   - "ANSI" C
# gnu89 - c89 plus GCC extensions
# c99   - ISO C99 standard (not yet fully implemented)
# gnu99 - c99 plus GCC extensions
#CSTANDARD = -std=gnu99
CDEBUG = -g$(DEBUG)
#CWARN = -Wall -Wstrict-prototypes
#CWARN = -Wall   # show all warnings
CWARN = -w      # suppress all warnings
####CTUNING = -funsigned-char -funsigned-bitfields -fpack-struct -fshort-enums
CTUNING = -ffunction-sections -fdata-sections
CXXTUNING = -fno-exceptions -ffunction-sections -fdata-sections
#CEXTRA = -Wa,-adhlns=$(<:.c=.lst)

CFLAGS = $(CDEBUG) -O$(OPT) $(CWARN) $(CTUNING) $(CDEFS) $(CINCS) $(CSTANDARD)
$(CEXTRA)
CXXFLAGS = $(CDEBUG) -O$(OPT) $(CWARN) $(CXXTUNING) $(CDEFS) $(CINCS)
#ASFLAGS = -Wa,-adhlns=$(<:.S=.lst),-gstabs
LDFLAGS = -O$(OPT) -lm -Wl,--gc-sections


# Programming support using avrdude. Settings and variables.
AVRDUDE_PORT = $(PORT)
AVRDUDE_WRITE_FLASH = -U flash:w:$(OUT_DIR)/$(TARGET).hex

#AVRDUDE_FLAGS = -V -F -C $(INSTALL_DIR)/hardware/tools/avr/etc/avrdude.conf \

AVRDUDE_FLAGS = -V -F -C $(INSTALL_DIR)/hardware/tools/avrdude.conf \
-p $(MCU) -P $(AVRDUDE_PORT) -c $(AVRDUDE_PROGRAMMER) \
-b $(UPLOAD_RATE)

# Program settings
CC = $(AVR_TOOLS_PATH)/avr-gcc
CXX = $(AVR_TOOLS_PATH)/avr-g++
LD = $(AVR_TOOLS_PATH)/avr-gcc
OBJCOPY = $(AVR_TOOLS_PATH)/avr-objcopy
OBJDUMP = $(AVR_TOOLS_PATH)/avr-objdump
AR  = $(AVR_TOOLS_PATH)/avr-ar
SIZE = $(AVR_TOOLS_PATH)/avr-size
NM = $(AVR_TOOLS_PATH)/avr-nm
AVRDUDE = $(AVRDUDE_PATH)/avrdude
REMOVE = rm -f
MV = mv -f

# Define all object files.
OBJ = $(SRC:.c=.o) $(CXXSRC:.cpp=.o) $(ASRC:.S=.o)
OBJ_MODULES = $(C_MODULES:.c=.o) $(CXX_MODULES:.cpp=.o)

# Define all listing files.
LST = $(ASRC:.S=.lst) $(CXXSRC:.cpp=.lst) $(SRC:.c=.lst)

# Combine all necessary flags and optional flags.
# Add target processor to flags.
ALL_CFLAGS = $(CFLAGS) -mmcu=$(MCU)
ALL_CXXFLAGS = $(CXXFLAGS) -mmcu=$(MCU)
ALL_ASFLAGS = -x assembler-with-cpp $(ASFLAGS) -mmcu=$(MCU)
ALL_LDFLAGS = $(LDFLAGS) -mmcu=$(MCU)

# Default target.
all: $(OUT_DIR)_files build sizeafter
build: elf hex

#$(OUT_DIR)_files: $(TARGET).pde
$(OUT_DIR)/$(TARGET).cpp: $(TARGET).pde
	test -d $(OUT_DIR) || mkdir $(OUT_DIR)
	echo '#include "Arduino.h"' > $(OUT_DIR)/$(TARGET).cpp
	echo 'void setup();' >> $(OUT_DIR)/$(TARGET).cpp
	echo 'void loop();' >> $(OUT_DIR)/$(TARGET).cpp
	cat $(TARGET).pde >> $(OUT_DIR)/$(TARGET).cpp

elf: $(OUT_DIR)/$(TARGET).elf
hex: $(OUT_DIR)/$(TARGET).hex
eep: $(OUT_DIR)/$(TARGET).eep
lss: $(OUT_DIR)/$(TARGET).lss
sym: $(OUT_DIR)/$(TARGET).sym

# Program the device.  
upload: $(OUT_DIR)/$(TARGET).hex
	$(AVRDUDE) $(AVRDUDE_FLAGS) $(AVRDUDE_WRITE_FLASH)


# Display size of file.
HEXSIZE = $(SIZE) --target=$(FORMAT) $(OUT_DIR)/$(TARGET).hex
ELFSIZE = $(SIZE)  $(OUT_DIR)/$(TARGET).elf
sizebefore:
	@if [ -f $(OUT_DIR)/$(TARGET).elf ]; then echo; echo $(MSG_SIZE_BEFORE); $(HEXSIZE); echo; fi

sizeafter:
	@if [ -f $(OUT_DIR)/$(TARGET).elf ]; then echo; echo $(MSG_SIZE_AFTER); $(HEXSIZE); echo; fi


# Convert ELF to COFF for use in debugging / simulating in AVR Studio or VMLAB.
COFFCONVERT=$(OBJCOPY) --debugging \
--change-section-address .data-0x800000 \
--change-section-address .bss-0x800000 \
--change-section-address .noinit-0x800000 \
--change-section-address .eeprom-0x810000


coff: $(OUT_DIR)/$(TARGET).elf
	$(COFFCONVERT) -O coff-avr $(OUT_DIR)/$(TARGET).elf $(TARGET).cof


extcoff: $(TARGET).elf
	$(COFFCONVERT) -O coff-ext-avr $(OUT_DIR)/$(TARGET).elf $(TARGET).cof


.SUFFIXES: .elf .hex .eep .lss .sym

.elf.hex:
	$(OBJCOPY) -O $(FORMAT) -R .eeprom $< $@

.elf.eep:
	$(OBJCOPY) -O $(FORMAT) -j .eeprom --set-section-flags=.eeprom="alloc,load" \
	--no-change-warnings \
	--change-section-lma .eeprom=0 $< $@

# Create extended listing file from ELF output file.
.elf.lss:
	$(OBJDUMP) -h -S $< > $@

# Create a symbol table from ELF output file.
.elf.sym:
	$(NM) -n $< > $@

	# Link: create ELF output file from library.
#$(OUT_DIR)/$(TARGET).elf: $(TARGET).pde $(OUT_DIR)/core.a
$(OUT_DIR)/$(TARGET).elf: $(OUT_DIR)/$(TARGET).o $(OUT_DIR)/core.a
	$(LD) $(ALL_LDFLAGS) -o $@ $(OUT_DIR)/$(TARGET).o $(OUT_DIR)/core.a

#$(OUT_DIR)/core.a: $(OBJ_MODULES) $(ARDUINO_MODULES)
$(OUT_DIR)/core.a: $(ARDUINO_MODULES) $(ARDUINO_CPP_MODULES)
	@for i in $(ARDUINO_MODULES) $(ARDUINO_CPP_MODULES); do echo $(AR) rcs $(OUT_DIR)/core.a $$i; $(AR) rcs $(OUT_DIR)/core.a $$i; done

$(ARDUINO_MODULES):
	$(CC) -c $(ALL_CFLAGS) $(ARDUINO)/$(patsubst %.o,%.c,$(patsubst $(OUT_DIR)/%,%,$@)) -o $@

$(ARDUINO_CPP_MODULES):
	$(CXX) -c $(ALL_CXXFLAGS) $(ARDUINO)/$(patsubst %.o,%.cpp,$(patsubst $(OUT_DIR)/%,%,$@)) -o $@

$(ARDUINO_CPP_LIBS):
	$(CXX) -c $(ALL_CXXFLAGS) $(ARDUINO_LIBS)/$(patsubst %.o,%.cpp,$(patsubst $(OUT_DIR)/%,%,$@)) -o $@

# Compile: create object files from C++ source files.
.cpp.o:
	$(CXX) -c $(ALL_CXXFLAGS) $< -o $@

# Compile: create object files from C source files.
.c.o:
	$(CC) -c $(ALL_CFLAGS) $< -o $@


# Compile: create assembler files from C source files.
.c.s:
	$(CC) -S $(ALL_CFLAGS) $< -o $@


# Assemble: create object files from assembler source files.
.S.o:
	$(CC) -c $(ALL_ASFLAGS) $< -o $@


# Automatic dependencies
%.d: %.c
	$(CC) -M $(ALL_CFLAGS) $< | sed "s;$(notdir $*).o:;$*.o $*.d:;" > $@

%.d: %.cpp
	$(CXX) -M $(ALL_CXXFLAGS) $< | sed "s;$(notdir $*).o:;$*.o $*.d:;" > $@


# Target: clean project.
clean:
	$(REMOVE) $(OUT_DIR)/$(TARGET).hex $(OUT_DIR)/$(TARGET).eep $(OUT_DIR)/$(TARGET).cof $(OUT_DIR)/$(TARGET).elf \
	$(OUT_DIR)/$(TARGET).map $(OUT_DIR)/$(TARGET).sym $(OUT_DIR)/$(TARGET).lss $(OUT_DIR)/core.a \
	$(OBJ) $(LST) $(SRC:.c=.s) $(SRC:.c=.d) $(CXXSRC:.cpp=.s) $(CXXSRC:.cpp=.d) $(ARDUINO_MODULES) $(ARDUINO_CPP_MODULES) $(OUT_DIR)/$(TARGET).cpp

.PHONY: all build elf hex eep lss sym program coff extcoff clean $(OUT_DIR)_files sizebefore sizeafter

#include $(SRC:.c=.d)
#include $(CXXSRC:.cpp=.d) 

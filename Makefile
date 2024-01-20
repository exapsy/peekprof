CC=go
CFLAGS=
LDFLAGS=
SOURCES=*.go
TARGET=peekprof
INSTALLDIR=~/.local/bin

.PHONY: all
all: clean $(TARGET) install

$(TARGET): $(SOURCES)
	$(CC) build -o $(TARGET) $(SOURCES)

.PHONY: clean
clean:
	@rm -f ./$(TARGET)

.PHONY: install
install: $(TARGET)
	@mkdir -p $(INSTALLDIR)
	@cp $(TARGET) $(INSTALLDIR)/$(TARGET)
	@echo "Installed $(TARGET) to $(INSTALLDIR)"

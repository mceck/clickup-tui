CC = go
BINDIR = bin
PREFIX = /usr/local
OUTPUT = clickup

TARGET = $(BINDIR)/$(OUTPUT)
SOURCES = cmd/clickup/main.go

all: $(BINDIR) $(TARGET)

$(BINDIR):
	mkdir -p $(BINDIR)

$(TARGET): $(SOURCES)
	$(CC) build -o $(TARGET) $(SOURCES)

install: $(TARGET)
	sudo install -d $(PREFIX)/bin
	sudo install -m 755 $(TARGET) $(PREFIX)/bin/$(OUTPUT)

uninstall:
	sudo rm -f $(PREFIX)/bin/$(OUTPUT)

clean:
	rm -rf $(BINDIR)

.PHONY: clean all

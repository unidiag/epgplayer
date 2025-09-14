APP := epgplayer-x86
TARGET_DIR := /EPG/epgby/frontend/public
TARGET := $(TARGET_DIR)/$(APP)

.PHONY: all clean

all:
	CGO_ENABLED=0 GOOS=linux GOARCH=386 GO386=softfloat \
		go build -tags netgo -trimpath -ldflags "-s -w" \
		-o ./$(APP)
	cp ./$(APP) $(TARGET)

clean:
	rm -f ./$(APP)
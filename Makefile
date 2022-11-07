GO  ?= go
APP := service

.PHONY: all
all:
	$(GO) build -o $(APP) ./cmd/$(APP)

.PHONY: clean
clean:
	$(GO) clean
	rm -f $(APP)

.PHONY: check
check: all
	$(GO) test -v ./...

.PHONY: run
run: all
	./$(APP)

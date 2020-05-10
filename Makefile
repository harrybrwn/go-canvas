install:
	go build ./canvas

uninstall:
	go clean -i ./canvas

clean:
	go clean

.PHONY: install uninstall clean


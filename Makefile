build:
	go build -o canvas-cli ./cli

install:
	go build -o canvas.bin ./cli
	mv ./canvas.bin $$GOPATH/bin/canvas

uninstall:
	go clean -i ./canvas
	$(RM) $$GOPATH/bin/canvas

clean:
	go clean

.PHONY: install uninstall clean


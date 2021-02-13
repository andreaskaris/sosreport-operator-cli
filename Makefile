fmt:
	go fmt ./...

build: clean fmt
	mkdir bin 2>/dev/null ; \
	go build -o bin/sosreport-operator-cli main.go

run: fmt
	go run main.go $(FLAGS)

clean:
	rm -f bin/*

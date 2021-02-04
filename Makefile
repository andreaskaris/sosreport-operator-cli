build:
	go build main.go -o bin/sosreport-operator-cli

run:
	go run main.go

clean:
	rm -f bin/*

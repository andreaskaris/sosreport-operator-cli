build: clean
	go build -o bin/sosreport-operator-cli main.go

run:
	go run main.go

clean:
	rm -f bin/*

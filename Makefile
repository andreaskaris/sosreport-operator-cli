build: clean
	mkdir bin 2>/dev/null ; \
	go build -o bin/sosreport-operator-cli main.go

run:
	go run main.go $(FLAGS)

clean:
	rm -f bin/*

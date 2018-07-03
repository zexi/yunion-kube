all: bin

bin: clean
	go build -o bin/kube-server ./cmd/kube-server/main.go

clean:
	rm -rf bin

.PHONY: all clean

all:
	go build -o wed ./cmd/wed

clean:
	rm wed

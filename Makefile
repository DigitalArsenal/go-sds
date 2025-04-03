# Makefile for local dev server with hot reload using reflex

APP_NAME := go-sds
ENTRY     := ./cmd
PORT      := 8080

.PHONY: all run dev kill build clean

all: run

run:
	go run $(ENTRY)

dev:
	@echo "Starting reflex with hot reload on port $(PORT)..."
	reflex -r '\.go$$' -s -- sh -c 'go run $(ENTRY)'

kill:
	@echo "Killing process on port $(PORT)..."
	-lsof -ti :$(PORT) | xargs kill -9 || true

build:
	go build -o bin/$(APP_NAME) $(ENTRY)

clean:
	rm -rf bin/$(APP_NAME)
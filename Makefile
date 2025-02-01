.PHONY: run
run:
	go run cmd/main.go

.PHONY: dev
dev:
	@if ! command -v air > /dev/null; then \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	air

.PHONY: build
build:
	go build -o bin/app cmd/main.go

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf proto/gen
	rm -rf tmp/

.PHONY: proto
proto:
	@echo "Generating proto files..."
	@for file in $$(find proto -name "*.proto" -not -path "proto/gen/*"); do \
		dir=$$(basename $$file .proto); \
		echo "Generating $$dir"; \
		mkdir -p proto/gen/$$dir; \
		protoc --proto_path=. \
			--go_out=. \
			--go_opt=module=github.com/elskow/chef-infra \
			--go-grpc_out=. \
			--go-grpc_opt=module=github.com/elskow/chef-infra \
			$$file; \
	done

.PHONY: setup
setup:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/air-verse/air@latest

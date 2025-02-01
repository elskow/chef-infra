.PHONY: run
run:
	APP_ENV=development go run cmd/main.go

.PHONY: run-prod
run-prod:
	APP_ENV=production go run cmd/main.go

.PHONY: run-test
run-test:
	APP_ENV=testing go run cmd/main.go

.PHONY: dev
dev:
	@if ! command -v air > /dev/null; then \
		echo "Installing air..."; \
		go install github.com/cosmtrek/air@latest; \
	fi
	APP_ENV=development	air

.PHONY: docker-dev
docker-dev:
	docker-compose -f docker-compose.dev.yaml up --build

.PHONY: docker-dev-down
docker-dev-down:
	docker-compose -f docker-compose.dev.yaml down

.PHONY: docker-dev-logs
docker-dev-logs:
	docker-compose -f docker-compose.dev.yaml logs -f

.PHONY: docker-clean
docker-clean:
	docker-compose -f docker-compose.dev.yaml down -v

.PHONY: build
build:
	go build -o bin/app cmd/main.go
	@mkdir -p bin/config
	@cp config/config.toml bin/config/

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf proto/gen
	rm -rf tmp/

.PHONY: proto
proto:
	@echo "Cleaning previous generated files..."
	@rm -rf proto/gen/*
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

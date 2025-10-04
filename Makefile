.PHONY: proto clean test run

# Configuration
DISH_ADDR ?= 192.168.100.1:9200
PROTO_DIR = proto
PROTO_OUT_DIR = $(PROTO_DIR)

# Fetch proto files from Starlink dish via gRPC reflection
fetch-protos:
	@echo "Fetching proto files from $(DISH_ADDR)..."
	@mkdir -p $(PROTO_OUT_DIR)
	grpcurl -plaintext -proto-out-dir $(PROTO_OUT_DIR) $(DISH_ADDR) describe SpaceX.API.Device.Device

# Add go_package option to proto files
fix-protos: fetch-protos
	@echo "Adding go_package options to proto files..."
	@find $(PROTO_DIR)/spacex_api -name "*.proto" -type f | while read -r file; do \
		if ! grep -q "option go_package" "$$file"; then \
			dir=$$(dirname "$$file" | sed 's|$(PROTO_DIR)/||'); \
			sed -i '1a\\noption go_package = "github.com/R167/starlink_exporter/proto/'$$dir'";' "$$file"; \
			echo "  Added go_package to $$file"; \
		fi \
	done

# Generate Go code from proto files
proto: fix-protos
	@echo "Generating Go code from proto files..."
	@find $(PROTO_DIR)/spacex_api -name "*.proto" -type f | while read -r file; do \
		protoc --proto_path=$(PROTO_DIR) \
			--go_out=$(PROTO_DIR) \
			--go_opt=paths=source_relative \
			"$$file"; \
	done
	@echo "Proto generation complete!"

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf $(PROTO_DIR)
	rm -f exporter

# Run tests
test:
	go test ./... -v

# Run the exporter
run:
	go run ./cmd/exporter

# Development: rebuild protos and run
dev: proto run

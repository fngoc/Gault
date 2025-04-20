# ====== Параметры TLS ======
CERT_DIR  := certs
CERT_CONF := $(CERT_DIR)/cert.conf

# ====== Параметры сборки клиента ======
VERSION     ?= 1.0.0
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
OUTPUT_BIN  := gault            # куда собирать клиент
CLIENT_PKG  := ./cmd/client     # корень main‑пакета

# ====== Цели по умолчанию ======
.PHONY: all tls proto mocks sqlc wire server client
all: tls proto mocks sqlc wire server client

# ---------- TLS ----------
tls: $(CERT_CONF)
	@echo "→ TLS certificates"
	mkdir -p $(CERT_DIR)
	openssl genrsa -out $(CERT_DIR)/ca.key 4096
	openssl req -x509 -new -nodes -key $(CERT_DIR)/ca.key \
		-sha256 -days 3650 \
		-out $(CERT_DIR)/ca.crt \
		-subj "/C=RU/ST=Some-State/L=Some-City/O=MyCompany/CN=MyRootCA"
	openssl genrsa -out $(CERT_DIR)/server.key 4096
	openssl req -new -key $(CERT_DIR)/server.key \
		-out $(CERT_DIR)/server.csr \
		-config $(CERT_CONF)
	openssl x509 -req -in $(CERT_DIR)/server.csr \
		-CA  $(CERT_DIR)/ca.crt \
		-CAkey $(CERT_DIR)/ca.key \
		-CAcreateserial \
		-out $(CERT_DIR)/server.crt \
		-days 3650 -sha256 \
		-extfile $(CERT_CONF) -extensions req_ext
	rm -f $(CERT_DIR)/server.csr $(CERT_DIR)/ca.srl

$(CERT_CONF):
	mkdir -p $(CERT_DIR)
	@printf '%s\n' \
'[ req ]' \
'default_bits = 4096' \
'distinguished_name = req_distinguished_name' \
'req_extensions = req_ext' \
'x509_extensions = req_ext' \
'prompt = no' \
'' \
'[ req_distinguished_name ]' \
'C  = RU' \
'ST = Some-State' \
'L  = Some-City' \
'O  = MyCompany' \
'CN = localhost' \
'' \
'[ req_ext ]' \
'subjectAltName = @alt_names' \
'' \
'[ alt_names ]' \
'DNS.1 = localhost' \
'IP.1  = 127.0.0.1' \
> $(CERT_CONF)

# ---------- Codegen ----------
proto:
	@echo "→ buf lint & generate"
	buf lint
	buf generate --path api/proto/v1

mocks:
	@echo "→ mockgen"
	mockgen -source=./internal/db/repository.go \
	        -destination=./gen/go/db/repository_mock.go \
	        -package=db

sqlc:
	@echo "→ sqlc generate"
	sqlc generate

wire:
	@echo "→ wire generate"
	cd internal/injector && wire

# ---------- Docker‑compose ----------
server:
	@echo "→ docker‑compose up"
	docker compose -f docker-compose.yml -p gault up -d

# ---------- Сборка клиента ----------
client:
	@echo "→ go build client (Version=$(VERSION), BuildDate=$(BUILD_DATE))"
	go build -ldflags "\
		-X 'main.Version=$(VERSION)' \
		-X 'main.BuildDate=$(BUILD_DATE)'" \
		-o $(OUTPUT_BIN) $(CLIENT_PKG)

CERT_DIR=certs
CERT_CONF=$(CERT_DIR)/cert.conf

.PHONY: tls
tls: $(CERT_CONF)
	mkdir -p $(CERT_DIR)
	openssl genrsa -out $(CERT_DIR)/ca.key 4096
	openssl req -x509 -new -nodes -key $(CERT_DIR)/ca.key -sha256 -days 3650 -out $(CERT_DIR)/ca.crt -subj "/C=RU/ST=Some-State/L=Some-City/O=MyCompany/CN=MyRootCA"
	openssl genrsa -out $(CERT_DIR)/server.key 4096
	openssl req -new -key $(CERT_DIR)/server.key -out $(CERT_DIR)/server.csr -config $(CERT_CONF)
	openssl x509 -req -in $(CERT_DIR)/server.csr -CA $(CERT_DIR)/ca.crt -CAkey $(CERT_DIR)/ca.key -CAcreateserial -out $(CERT_DIR)/server.crt -days 3650 -sha256 -extfile $(CERT_CONF) -extensions req_ext
	rm $(CERT_DIR)/server.csr $(CERT_DIR)/ca.srl

$(CERT_CONF):
	mkdir -p $(CERT_DIR)
	echo "[ req ]\n\
default_bits = 4096\n\
distinguished_name = req_distinguished_name\n\
req_extensions = req_ext\n\
x509_extensions = req_ext\n\
prompt = no\n\
\n\
[ req_distinguished_name ]\n\
C = RU\n\
ST = Some-State\n\
L = Some-City\n\
O = MyCompany\n\
CN = localhost\n\
\n\
[ req_ext ]\n\
subjectAltName = @alt_names\n\
\n\
[ alt_names ]\n\
DNS.1 = localhost\n\
IP.1 = 127.0.0.1" > $(CERT_CONF)

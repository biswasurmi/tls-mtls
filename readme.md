---

# TLS + mTLS Setup Guide

This guide shows how to generate a Certificate Authority (CA), server certificate, and client certificate for testing TLS and mutual TLS (mTLS) with a Go server.

---

## 📂 Setup Working Directory

```bash
mkdir -p certs && cd certs
```

---

## 🔑 Create a Certificate Authority (CA)

```bash
mkdir ca
cd ca

# Generate CA private key
openssl genrsa -out ca.key 4096
chmod 600 ca.key

# Create ca.ext file
cat > ca.ext <<EOF
[ v3_ca ]
basicConstraints = critical, CA:true
keyUsage = critical, keyCertSign, cRLSign
subjectKeyIdentifier = hash
EOF

# Generate CA certificate
openssl req -x509 -new -nodes -days 3650 -sha256 \
  -key ca.key -out ca.crt \
  -subj "/CN=backup.local/O=kubedb" \
  -extensions v3_ca -extfile ca.ext
```

Verify:

```bash
openssl x509 -in ca.crt -noout -text | grep "CA:TRUE"
```

---

## 🖥️ Create Server Certificate

```bash
cd ../
mkdir server
cd server

# Generate server key and CSR
openssl req -newkey rsa:2048 -nodes \
  -keyout server.key -out server.csr \
  -subj "/CN=backup.local"
chmod 600 server.key

# Create server.ext file
cat > server.ext <<EOF
[ v3_req ]
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = backup.local
IP.1  = 127.0.0.1
EOF

# Sign server cert with CA
openssl x509 -req -in server.csr -CA ../ca/ca.crt -CAkey ../ca/ca.key -CAcreateserial \
  -out server.crt -days 825 -sha256 \
  -extensions v3_req -extfile server.ext
```

Verify:

```bash
openssl verify -CAfile ../ca/ca.crt server.crt
openssl x509 -in server.crt -noout -text | grep -A2 "Subject Alternative Name"
```

---

## 👤 Create Client Certificate (for mTLS)

```bash
cd ../
mkdir client
cd client

# Generate client key and CSR
openssl req -newkey rsa:2048 -nodes \
  -keyout client.key -out client.csr \
  -subj "/CN=clients"
chmod 600 client.key

# Create client.ext file
cat > client.ext <<EOF
[ v3_req ]
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF

# Sign client cert with CA
openssl x509 -req -in client.csr -CA ../ca/ca.crt -CAkey ../ca/ca.key -CAcreateserial \
  -out client.crt -days 825 -sha256 \
  -extensions v3_req -extfile client.ext
```

Verify:

```bash
openssl verify -CAfile ../ca/ca.crt client.crt
```

---

## 🚀 Run Go Server with TLS / mTLS

### server.go (basic TLS → mTLS)

```go
package main

import (
    "crypto/tls"
    "crypto/x509"
    "fmt"
    "io/ioutil"
    "net/http"
)

func main() {
    // Load CA cert to verify client certs
    caCert, _ := ioutil.ReadFile("ca/ca.crt")
    caCertPool := x509.NewCertPool()
    caCertPool.AppendCertsFromPEM(caCert)

    mux := http.NewServeMux()
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello, mTLS world!")
    })

    srv := &http.Server{
        Addr:    ":8443",
        Handler: mux,
        TLSConfig: &tls.Config{
            MinVersion: tls.VersionTLS12,
            ClientAuth: tls.RequireAndVerifyClientCert,
            ClientCAs:  caCertPool,
        },
    }

    fmt.Println("Starting mTLS server on https://backup.local:8443")
    panic(srv.ListenAndServeTLS("server/server.crt", "server/server.key"))
}
```

Run:

```bash
go run main.go
```

---

## 🧪 Test with curl

### Without client cert (❌ rejected)

```bash
curl https://backup.local:8443 --cacert ./ca/ca.crt --resolve backup.local:8443:127.0.0.1
```

### With client cert (✅ success)

```bash
curl https://backup.local:8443 \
  --cacert ./ca/ca.crt \
  --cert ./client/client.crt --key ./client/client.key \
  --resolve backup.local:8443:127.0.0.1
```

Expected:

```
Hello, mTLS world!
```

---

## Project Structure

```bash
.
├── ca
│   ├── ca.crt
│   ├── ca.key
│   └── ca.srl
├── client
│   ├── client.crt
│   ├── client.csr
│   ├── client.ext
│   └── client.key
├── readme.md
├── server
│   ├── server.crt
│   ├── server.csr
│   ├── server.ext
│   └── server.key
└── server.go

4 directories, 13 files
```

---

## 📝 Summary

* `ca/ca.crt` / `ca/ca.key`: Root CA
* `server/server.crt` / `server/server.key`: Server certificate
* `client/client.crt` / `client/client.key`: Client certificate (for mTLS)
* `server.ext` & `client.ext`: Extension config files

---



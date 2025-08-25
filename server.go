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
    caCert, err := ioutil.ReadFile("ca.crt")
    if err != nil {
        panic(err)
    }

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
            // Require client certs signed by our CA
            ClientAuth: tls.RequireAndVerifyClientCert,
            ClientCAs:  caCertPool,
        },
    }

    fmt.Println("Starting mTLS server on https://backup.local:8443")
    err = srv.ListenAndServeTLS("server.crt", "server.key")
    if err != nil {
        panic(err)
    }
}

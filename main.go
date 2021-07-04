package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    http.HandleFunc("/", HelloHandler)
    http.HandleFunc("/healthcheck", HealthCheckHandler)
    log.Println("Server started. Listening on port 8080.")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func HelloHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello")
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "OK")
}
package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type key int

const (
	requestIDKey key = 0
)

var (
	listenAddr string
	trustedIP  string
	healthy    int32
	Version    string = "dev"
)

type renderContext struct {
	Headers   map[string]string
	Version   string
	RequestID string
	CallerIP  string
}

func main() {
	flag.StringVar(&listenAddr, "binding", "0.0.0.0:3000", "Server listen address")
	flag.StringVar(&trustedIP, "trusted-ip", "", "Load balancer IP to trust")
	flag.Parse()

	cancel := make(chan os.Signal)
	signal.Notify(cancel, os.Interrupt, syscall.SIGTERM)

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Printf("Server is starting on %s...\n", listenAddr)

	router := http.NewServeMux()
	router.Handle("/favicon.ico", http.FileServer(http.Dir("./static")))
	router.HandleFunc("/", handler)

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      logging(logger)(router),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	logger.Println("Server stopped")
}

func handler(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	w.Header().Set("X-Request-Id", requestID)

	var err error
	templ, err := template.New("index.html").ParseFiles("./static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for name, values := range r.Header {
		for _, value := range values {
			fmt.Println(name, value)
		}
	}

	headers := make(map[string]string)

	for name, values := range r.Header {
		headers[name] = strings.Join(values, ",")
	}

	// get the real IP from the header if present and only trust the given IPs

	err = templ.Execute(w, renderContext{
		Headers:   headers,
		Version:   Version,
		RequestID: requestID,
		CallerIP:  r.RemoteAddr,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

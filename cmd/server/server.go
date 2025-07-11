package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jdecool/finalurl/checker"
)

const (
	defaultPort         = 8080
	requestTimeout      = 30 * time.Second
	shutdownTimeout     = 15 * time.Second
	maxURLLength        = 2048
	defaultReadTimeout  = 10 * time.Second
	defaultWriteTimeout = 10 * time.Second
	defaultIdleTimeout  = 60 * time.Second
)

type Server struct {
	templates *template.Template
	checker   *checker.Checker
}

func NewServer() (*Server, error) {
	tmpl, err := template.ParseFiles("./cmd/server/template/check.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &Server{
		templates: tmpl,
		checker:   &checker.Checker{CheckRobotTxt: true},
	}, nil
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %v %s",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start),
			r.RemoteAddr,
		)
	})
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) timeoutMiddleware(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, requestTimeout, "Request timeout")
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *Server) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	location := strings.TrimSpace(r.URL.Query().Get("url"))
	if location == "" {
		http.Error(w, "Missing required parameter: url", http.StatusBadRequest)
		return
	}

	if len(location) > maxURLLength {
		http.Error(w, "URL too long", http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(location); err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	parsedURL, _ := url.Parse(location)
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		http.Error(w, "Only HTTP and HTTPS URLs are supported", http.StatusBadRequest)
		return
	}

	disableRobotsTxt := r.URL.Query().Get("disable-robotstxt") != ""

	c := &checker.Checker{
		CheckRobotTxt: !disableRobotsTxt,
	}

	flow, err := c.GetRedirections(location)
	if err != nil {
		log.Printf("Error checking URL %s: %v", location, err)
		http.Error(w, fmt.Sprintf("Failed to check URL: %v", err), http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	acceptHeader := r.Header.Get("Accept")
	wantsJSON := format == "json" || strings.Contains(acceptHeader, "application/json")

	if wantsJSON {
		s.respondWithJSON(w, flow)
	} else {
		s.respondWithHTML(w, flow)
	}
}

func (s *Server) respondWithJSON(w http.ResponseWriter, flow checker.Flow) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(flow); err != nil {
		log.Printf("JSON encoding error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) respondWithHTML(w http.ResponseWriter, flow checker.Flow) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := s.templates.Execute(w, flow); err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","service":"finalurl-server"}`))
}

func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./cmd/server/static"))
	mux.Handle("/", s.loggingMiddleware(fileServer))

	mux.HandleFunc("/check", s.handleCheck)
	mux.HandleFunc("/health", s.handleHealth)

	return mux
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	mux := server.setupRoutes()

	handler := server.corsMiddleware(server.loggingMiddleware(server.timeoutMiddleware(mux)))

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", defaultPort),
		Handler:      handler,
		ReadTimeout:  defaultReadTimeout,
		WriteTimeout: defaultWriteTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting server on port %d", defaultPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	} else {
		log.Println("Server gracefully stopped")
	}
}

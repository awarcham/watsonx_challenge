package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"text/template"
	"time"
)

type GitHubIssue struct {
	IssueNumber  int                    `json:"number"`
	Title        int                    `json:"title"`
	Body         int                    `json:"body"`
	CreationDate time.Time              `json:"created_at"`
	ClosedDate   time.Time              `json:"closed_at"`
	ClosedBy     map[string]interface{} `json:"closed_by"`
}

func main() {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	srv := startServer(wg, createServerMux())
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Kill, os.Interrupt)
	<-sigChan
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("Error shutting down web server: %v", err)
		wg.Done()
	}
	wg.Wait()
	log.Println("Web server exited")
	os.Exit(0)
}

func createServerMux() *http.ServeMux {
	ret := http.NewServeMux()
	issueTemplate, tmplErr := template.New("issue").Parse("## Issue {{.IssueNumber}}: {{.Title}}\n**Created**: {{.CreationDate}}\n**Closed**: {{.ClosedDate}}\n**Closed By**: {{.ClosedBy.login}}\n### Description\n{{.Body}}\n")
	if tmplErr != nil {
		log.Fatalf("Invalid issue template: %v", tmplErr)
	}
	ret.Handle("/markdown", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var outFile strings.Builder
		outFile.WriteString("# Release Notes")

		method := r.Method
		if method != "POST" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Only POST requests are allowed to this endpoint"))
			return
		}
		var issues []GitHubIssue
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Request body must not be empty"))
			return
		}
		log.Printf("Request body: %v", string(body))
		err = json.Unmarshal(body, &issues)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("Error reading request body: %v", err)))
			log.Printf("%v", err)
			return
		}
		for issue := range issues {
			issueTemplate.Execute(&outFile, issue)
		}
		out := outFile.String()
		var outBin string
		for _, c := range out {
			outBin = fmt.Sprintf("%s%b", outBin, c)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(outBin))
	}))
	return ret
}

func startServer(wg *sync.WaitGroup, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:    "localhost:80",
		Handler: handler,
	}
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Error from HTTP server: %v", err)
		}
	}()
	return srv
}

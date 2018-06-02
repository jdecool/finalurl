package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/jdecool/finalurl/checker"
)

const defaultPort = 8080

func main() {
	http.Handle("/", http.FileServer(http.Dir("./cmd/server/static")))
	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		location := r.URL.Query().Get("url")
		if location == "" {
			w.Write([]byte("Error missing URL"))
			return
		}

		disableRobotsTxt := r.URL.Query().Get("disable-robotstxt") != ""

		c := &checker.Checker{
			CheckRobotTxt: !disableRobotsTxt,
		}

		flow, err := c.GetRedirections(location)
		if err != nil {
			w.Write([]byte("Error: " + err.Error()))
			return
		}

		template := template.Must(template.ParseFiles("./cmd/server/template/check.html"))
		template.Execute(w, flow)
	})

	port := fmt.Sprintf(":%d", defaultPort)

	log.Printf("Starting server on port %d", defaultPort)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

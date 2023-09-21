package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"os"
)

func main() {

	oauthServer := os.Getenv("OAUTH_SERVER")
	if len(oauthServer) == 0 {
		log.Fatal("No auth server configured! Set OAUTH_SERVER")
	}

	server := os.Getenv("SERVER")
	if len(server) == 0 {
		log.Fatal("No server configured! Set OAUTH_SERVER")
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		original, err := url.Parse(query.Get("redirect_uri"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		reduced := &url.URL{
			Scheme: "https",
			Host:   original.Host,
			Path:   original.Path,
		}

		base64 := base64.StdEncoding.EncodeToString([]byte(reduced.String()))
		query.Set("redirect_uri", server)

		redirect := &url.URL{
			Scheme:   "https",
			Host:     server,
			Path:     "/back/" + base64,
			RawQuery: query.Encode(),
		}

		w.Header().Add("Location", redirect.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.HandleFunc("/back/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		log.Print(r.URL.Path)
		if len(parts) != 3 {
			log.Print("no there paths")
			w.WriteHeader(http.StatusBadRequest)
		}

		urlString, err := base64.StdEncoding.DecodeString(parts[2])
		if err != nil {
			log.Print("could not decode url")
			w.WriteHeader(http.StatusBadRequest)
		}
		log.Print(urlString)
		original, err := url.Parse(string(urlString))
		if err != nil {
			log.Print("could not parse url")
			w.WriteHeader(http.StatusBadRequest)
		}
		redirect := &url.URL{
			Scheme:   "https",
			Host:     original.Host,
			Path:     original.Path,
			RawQuery: r.URL.RawQuery,
		}
		log.Print(redirect.String())
		w.Header().Add("Location", redirect.String())

	})
	fmt.Println("Hello, World!")
	err := http.ListenAndServe(":3000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("server closed")
	} else if err != nil {
		fmt.Printf("error starting server: %s", err)
		os.Exit(1)
	}
}

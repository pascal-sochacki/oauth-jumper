package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"os"
)

func getRedirectUrl(original *url.URL) string {
	base64 := base64.StdEncoding.EncodeToString([]byte(original.String()))
	comebackAt := &url.URL{
		Scheme: "http",
		Host:   "localhost:3001",
		Path:   "back/" + base64,
	}

	result := comebackAt.String()
	log.Println("redirec uri", result)
	return result
}

func main() {

	oauthServer := os.Getenv("OAUTH_SERVER")
	if len(oauthServer) == 0 {
		log.Fatal("No auth server configured! Set OAUTH_SERVER")
	}

	server := os.Getenv("SERVER")
	if len(server) == 0 {
		log.Fatal("No server configured! Set OAUTH_SERVER")
	}

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}

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
			Scheme:   "http",
			Host:     original.Host,
			Path:     original.Path,
			RawQuery: r.URL.RawQuery,
		}
		log.Print(redirect.String())
		w.Header().Add("Location", redirect.String())
		w.WriteHeader(http.StatusTemporaryRedirect)

	})
	http.HandleFunc("/realms/apps-cc/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)
		url := &url.URL{
			Scheme: "http",
			Host:   oauthServer,
			Path:   r.URL.Path,
		}
		log.Println("request", url.String())
		res, err := http.Get(url.String())
		if err != nil {
			log.Println("Got error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println(res.StatusCode)
		defer res.Body.Close()
		for key, values := range res.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(res.StatusCode)

		var responseData map[string]interface{}
		decoder := json.NewDecoder(res.Body)
		if err := decoder.Decode(&responseData); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		val, ok := responseData["authorization_endpoint"].(string)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		auth, err := url.Parse(val)
		auth.Host = "localhost:3001"
		responseData["authorization_endpoint"] = auth.String()

		val, ok = responseData["token_endpoint"].(string)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		auth, err = url.Parse(val)
		auth.Host = "localhost:3001"
		responseData["token_endpoint"] = auth.String()

		_, copyErr := io.Copy(w, res.Body)
		if copyErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		modifiedJSON, err := json.Marshal(responseData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(modifiedJSON)
	})
	http.HandleFunc("/realms/apps-cc/protocol/openid-connect/auth", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		original, err := url.Parse(query.Get("redirect_uri"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		reduced := &url.URL{
			Scheme: "http",
			Host:   original.Host,
			Path:   original.Path,
		}

		query.Set("redirect_uri", getRedirectUrl(reduced))

		redirect := &url.URL{
			Scheme:   "http",
			Host:     oauthServer,
			Path:     r.URL.Path,
			RawQuery: query.Encode(),
		}

		w.Header().Add("Location", redirect.String())
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	http.HandleFunc("/realms/apps-cc/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Query())
		log.Println(r.Header)
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		url := &url.URL{
			Scheme: "http",
			Host:   oauthServer,
			Path:   r.URL.Path,
		}
		original, err := url.Parse(r.Form.Get("redirect_uri"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		r.Form.Set("redirect_uri", getRedirectUrl(original))
		req, err := http.NewRequest("POST", url.String(), strings.NewReader(r.Form.Encode()))
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		res, err := httpClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}

		log.Println(r.Form)
		log.Println("Get Token called!")

		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		defer res.Body.Close()
		for key, values := range res.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		log.Println(string(bytes))
		w.WriteHeader(res.StatusCode)
		w.Write(bytes)
		return
	})
	err := http.ListenAndServe(":3001", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Println("server closed")
	} else if err != nil {
		fmt.Printf("error starting server: %s", err)
		os.Exit(1)
	}
}

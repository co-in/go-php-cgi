package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
)

func fileExist(basePath string, uri string) bool {
	if uri == "/" {
		return false
	}

	info, err := os.Stat(basePath + uri)

	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func readConfig(fileName string, response interface{}) error {
	configFile, err := os.Open(fileName)

	if err != nil {
		return err
	}

	defer func() {
		_ = configFile.Close()
	}()

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(response)

	if err != nil {
		return err
	}

	return nil
}

type indexRoot struct {
	Root  string
	Index string
}

type config struct {
	INI     string
	CGI     string
	Port    string
	Headers map[string]string
	Route   map[string]indexRoot
}

var configPath = flag.String("config", "config.json", "Config file location")

func main() {
	var err error

	flag.Parse()
	config := new(config)

	err = readConfig(*configPath, config)

	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var ir indexRoot
		var ok bool

		if ir, ok = config.Route[r.Host]; !ok {
			//Route not configured
			return
		}

		var host = ir.Root
		var index = ir.Index

		for k, v := range config.Headers {
			w.Header().Set(k, v)
		}

		if fileExist(host, r.URL.Path) {
			http.ServeFile(w, r, host+r.URL.Path)
		} else {
			host = host + "/" + index
			handler := new(cgi.Handler)
			handler.Path = config.CGI
			handler.Env = append(handler.Env, "REDIRECT_STATUS=CGI")
			handler.Env = append(handler.Env, "SCRIPT_NAME="+host)
			handler.Env = append(handler.Env, "SCRIPT_FILENAME="+host)
			handler.Env = append(handler.Env, "HOST="+r.Host)

			for k, v := range r.Header {
				if len(v) > 1 {
					fmt.Println("ERROR", v)
				}

				handler.Env = append(handler.Env, k+"="+v[0])
			}

			handler.Env = append(handler.Env, "PHPRC="+config.INI)
			handler.Env = append(handler.Env, "PHP_CGI="+config.CGI)

			handler.ServeHTTP(w, r)
		}
	})

	err = http.ListenAndServe(config.Port, nil)

	if err != nil {
		log.Fatal(err)
	}
}

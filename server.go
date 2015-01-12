/* Copyright (C) Pat Kaehuaea - All Rights Reserved
 * Unauthorized copying of this file, via any medium is strictly prohibited
 * Proprietary and confidential
 * Written by Pat Kaehuaea, January 2015
 */

 // TODO: Extend Cookie Struct

package main

import (
 	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/Sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const COOKIE_NAME = "uuid"

// credit: http://stackoverflow.com/questions/17206467/go-how-to-render-multiple-templates-in-golang
var templates = template.Must(template.ParseGlob(htmlTemplPath()))

// credit: https://blog.golang.org/go-maps-in-action
var users = struct{
    sync.RWMutex
    m map[string]string
}{m: make(map[string]string)}

func htmlTemplPath() string {
	curDir, _ := os.Getwd()
	templatesPath := filepath.Join(curDir, "templates", "*.html")
	return templatesPath
}

func getTime() string {
	const layout = "3:04:05 PM"
	t := time.Now().Format(layout)
	return t
}

func uuid() string {
	// credit: http://golang.org/pkg/os/exec/#Cmd.Run
	log.Debug("Getting uuid.")
	out, err := exec.Command("/usr/bin/uuidgen").Output()
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("Removing trailing '\n' from output. ")
	uuid := strings.TrimSuffix(string(out), "\n")

	log.Debug("Output of uuid generator: " + uuid)
	return uuid
}

// credit: https://blog.golang.org/go-maps-in-action
func findName(uuid string) string {
	users.RLock()
	name := users.m[uuid]
	users.RUnlock()
	return name
}

// credit: https://blog.golang.org/go-maps-in-action
func addName(uuid string, name string) bool {
	if validateName(name) {
		users.Lock()
		users.m[uuid] = name
		users.Unlock()
		return true
	}
	return false
}

func validateName(name string) bool {
	// TODO: better to implement in form?
	// TODO: implement name validation
	return true
}

func uuidToName(r *http.Request) (uName string, err error) {
	log.Debug("Reading cookie 'uuid' and finding name.")

	cookie, err := r.Cookie(COOKIE_NAME)
	// TODO: Implement additional cookie validation
	// like domain and expiry in own method.
	if err == http.ErrNoCookie {
		return "", http.ErrNoCookie
	}

	name := findName(cookie.Value)
	
	if name == "" {
        return "", errors.New("Cookie value not found in user table.")
	}

	return name, nil
}

func setCookie(w http.ResponseWriter, uuid string) {
	c := http.Cookie {Name: COOKIE_NAME, Value: uuid, Path: "/"}
	http.SetCookie(w, &c)
}

// Duplicate cookies are not handled as it should not be possible as
// login with name sets cookie and overwrites.
func deleteCookie(w http.ResponseWriter) {
	log.Debug("Deleting cookie.")
	// Invalidate data along and set MaxAge to avoid accidental persistence issues.
	c := http.Cookie {Name: COOKIE_NAME, Value: "deleted", Path: "/", MaxAge: -1}
	http.SetCookie(w, &c)
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Default handler called.", r)
	
	name, err := uuidToName(r)
	if name == "" || err != nil {
		log.Debug("No cookie found or value empty. Redirecting to login.")
		http.Redirect(w, r, "/login", http.StatusFound)
	}

	log.Debug("Cookie uuid found in user table: " + name)
	templates.ExecuteTemplate(w, "greetings", name)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Login handler called.")

	if(r.Method == "GET") {
		templates.ExecuteTemplate(w, "login", nil)
		log.Debug("Login template rendered.")
	}

	if(r.Method == "POST") {
		log.Debug("POST method detected.")
		// Form will not submit if name empty.
		name := r.FormValue("name")
		if validateName(name) {
			uuid := uuid()
			addName(uuid, name)
			setCookie(w, uuid)
			http.Redirect(w, r, "/", http.StatusFound)
	        return
		} else {
			// Redirect user with 4xx status code.
			log.Debug("Invalid username. Redirecting to root.")
			w.WriteHeader(http.StatusBadRequest )
			http.Redirect(w, r, "/", http.StatusFound)
		}
	}

	log.Debug("Request method not handled.")
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Logout handler called.")
	deleteCookie(w)
	templates.ExecuteTemplate(w, "logged-out.html", nil)
}

func timeHandler(w http.ResponseWriter, r *http.Request) {
	logInfo("Time handler called.", r)
	name, _ := uuidToName(r)
	// No error checking for name since logic implemented
	// in template.
	params := map[string]interface{}{"time": getTime(), "name": name}
	templates.ExecuteTemplate(w, "time", params)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	logInfo("Not found handler called.", r)
	w.WriteHeader(http.StatusNotFound)
	templates.ExecuteTemplate(w, "404", nil)
}

func logInfo(msg string, r *http.Request) {
	log.WithFields(log.Fields{
		"method": r.Method,
		"time": getTime(),
		"url": r.URL,
	}).Info(msg)
}

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}
func main() {

	const VERSION_NUMBER = "v1.0.3"

	portPtr := flag.String("port", "8080", "Web server binds to this port. Default is 8080.")
	verbosePtr := flag.Bool("V", false, "Prints version number of program.")
	portString := ":" + *portPtr
	flag.Parse()

	if *verbosePtr {
		fmt.Printf("Version number: %s \n", VERSION_NUMBER)
		os.Exit(1)
	}

	//credit: http://stackoverflow.com/questions/9996767/showing-custom-404-error-page-with-standard-http-package
	r := mux.NewRouter()
	r.HandleFunc("/", defaultHandler)
	r.HandleFunc("/index.html", defaultHandler)
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/logout", logoutHandler)
	r.HandleFunc("/time", timeHandler)
	r.NotFoundHandler = http.HandlerFunc(notFound)
	http.Handle("/", r)
	http.ListenAndServe(portString, nil)
}

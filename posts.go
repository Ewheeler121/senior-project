package main

import "net/http"

func submitPageHandler(w http.ResponseWriter, r *http.Request) {
    renderTemplate(w, r, "submit.html", "Submit", nil) 
}

func summitPostHandler(w http.ResponseWriter, r *http.Request) {
    
}

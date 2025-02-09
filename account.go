package main

import (
	"net/http"

	"golang.org/x/crypto/bcrypt"

	_ "github.com/mattn/go-sqlite3"
)

func getUser(r *http.Request) string {
    session, _ := cookies.Get(r, "session")
    user := session.Values["userID"]
    if user != nil {
        return user.(string)
    }
    return ""
}

func Auth(HandlerFunc http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if getUser(r) == "" {
			http.Redirect(w, r, "/login", 302)
			return
		}
		HandlerFunc.ServeHTTP(w, r)
	}
}

func loginPageHandler(w http.ResponseWriter, r *http.Request) {
    renderTemplate(w, r, "login.html", "Login", nil)
}

func registerPageHandler(w http.ResponseWriter, r *http.Request) {
    renderTemplate(w, r, "register.html", "Register", nil)
}

func loginAuthHandler(w http.ResponseWriter, r *http.Request) {
    failed := "username or password are incorrect"

    err := r.ParseForm()
    if err != nil {
        http.NotFound(w, r)
        return
    }

    username := r.FormValue("username")
    password := r.FormValue("password")

    var userID, hash string
    row := db.QueryRow("SELECT username, password FROM users WHERE username = ?", username)
    err = row.Scan(&userID, &hash)
    if err != nil {
        renderTemplate(w, r, "login.html", "Login", map[string]interface{}{"message": failed})
        return
    }

    err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    if err != nil {
        renderTemplate(w, r, "login.html", "Login", map[string]interface{}{"message": failed})
        return
    }

    session, _ := cookies.Get(r, "session")
    session.Values["userID"] = userID
    session.Save(r, w)
    indexHandler(w, r)
}

func logoutAuthHandler(w http.ResponseWriter, r *http.Request) {
    session, _ := cookies.Get(r, "session")
    delete(session.Values, "userID")
    session.Save(r, w)
    indexHandler(w, r)
}


//TODO: add email verification
//https://www.youtube.com/watch?v=guDfl9oqN-I&list=PLDZ_9qD1hkzOQdLHOPHtDcxoDSr0nno9G&index=22
func registerAuthHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        http.NotFound(w, r)
        return
    }

    username := r.FormValue("username")
    email := r.FormValue("email")
    password := r.FormValue("password")
    
    //TODO: add password requirements + email validation here

    _, err = db.Query("SELECT username FROM users WHERE username = ? or email = ?", username, email)
    if err != nil {
        renderTemplate(w, r, "register.html", "Register", map[string]interface{}{"message": "Username/email already exists"})
        return
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        renderTemplate(w, r, "register.html", "Register", map[string]interface{}{"message": "there was a problem registering account"})
        return
    }
    
    mu.Lock()
    defer mu.Unlock()
    insertStmt, err := db.Prepare("INSERT INTO users (username, email, password) VALUES (?, ?, ?)")
    if err != nil {
        renderTemplate(w, r, "register.html", "Register", map[string]interface{}{"message": "there was a problem registering account"})
        return
    }
    defer insertStmt.Close()

    _, err = insertStmt.Exec(username, email, hash)
    if err != nil {
        renderTemplate(w, r, "register.html", "Register", map[string]interface{}{"message": "there was a problem registering account"})
        return
    }

    renderTemplate(w, r, "login.html", "Login", map[string]interface{}{"message": "account created sucessfully"})
}

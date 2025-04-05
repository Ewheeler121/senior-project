package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/context"
	"github.com/gorilla/sessions"

	_ "github.com/mattn/go-sqlite3"
)

type tplData = map[string]interface{}

var tpl *template.Template

var db *sql.DB
var mu sync.Mutex

var cookies *sessions.CookieStore

func main() {
	debugPrint("SESSION_KEY: ", os.Getenv("SESSION_KEY"))
	debugPrint("API_KEY: ", os.Getenv("API_KEY"))

	if os.Getenv("SESSION_KEY") == "" {
		panic("environment variable SESSION_KEY is not set, run\nexport SESSION_KEY={session key}")
	}

	if os.Getenv("API_KEY") == "" {
		panic("environment variable API_KEY is not set, run\nexport API_KEY={api key}")
	}

	init_database()
	defer db.Close()

	tpl = template.Must(template.ParseGlob("templates/*.html"))
	cookies = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	http.HandleFunc("/loginauth", loginAuthHandler)
	http.HandleFunc("/login", loginPageHandler)
	http.HandleFunc("/logout", auth(logoutAuthHandler))
	http.HandleFunc("/registerauth", registerAuthHandler)
	http.HandleFunc("/register", registerPageHandler)
	http.HandleFunc("/profile", profileHandler)

	http.HandleFunc("/submit", auth(submitPageHandler))
	http.HandleFunc("/submitPost", auth(submitPostHandler))
	http.HandleFunc("/download/", entryDownloadHandler)
	http.HandleFunc("/entry/", entryPageHandler)
	http.HandleFunc("/edit/", auth(editEntryHandler))
	http.HandleFunc("/delete/", deleteEntryHandler)
	http.HandleFunc("/editEntry/", auth(editEntryPostHandler))
	http.HandleFunc("/addFile/", auth(addFileHandler))
	http.HandleFunc("/replaceFile/", auth(replaceFileHandler))
	http.HandleFunc("/deleteFile/", auth(deleteFileHandler))

	http.HandleFunc("/searchPost", searchPostHandler)
	http.HandleFunc("/search", searchPageHandler)

	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)
	http.HandleFunc("/", nullHandler)

	//TODO: remove this line after testing
	err := http.ListenAndServeTLS(":443", "domain.cert.pem", "private.key.pem", context.ClearHandler(http.DefaultServeMux))
	//err := http.ListenAndServe("localhost:8080", context.ClearHandler(http.DefaultServeMux))
	if err != nil {
		debugPrint("cannot start server ", err)
		fmt.Fprintln(os.Stderr, "ERROR: could not start server, ", err)
	}
}

func nullHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path[len("/"):] == "" {
		indexHandler(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, ".css") {
		http.ServeFile(w, r, "static/css"+r.URL.Path)
		return
	}
	if strings.HasSuffix(r.URL.Path, ".webp") || strings.HasSuffix(r.URL.Path, ".ico") {
		http.ServeFile(w, r, "static/img"+r.URL.Path)
		return
	}
	http.NotFound(w, r)
}

func init_database() {
	var err error
	db, err = sql.Open("sqlite3", "./database.db")
	if err != nil {
		panic(err.Error())
	}

	//TODO: add more to users for profile page
	//TODO: modify Entries table to include more categories etc..
	//TODO: modify Files table to include more categories
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            username TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL,
            email TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS entries (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            title TEXT NOT NULL,
            submitted TEXT NOT NULL,
            authors TEXT NOT NULL,
            gradlevel TEXT CHECK(gradlevel IN (
				'Highschool',
				'Undergraduate',
				'Graduate'
			)) NOT NULL,
            affiliation TEXT NOT NULL,
            keywords TEXT,
            abstract TEXT,
            comments TEXT,
            category TEXT CHECK(category IN (
                'Computer Science', 
                'Physics', 
                'Mathematics', 
                'Engineering', 
                'Biology'
            )) NOT NULL,
            license TEXT CHECK (license IN (
                'CC BY',
                'CC BY-SA',
                'CC BY-ND',
                'CC BY-NC',
                'CC BY-NC-SA',
                'CC BY-NC-ND',
                'MIT',
                'GPLv3',
                'Apache 2.0',
                'Unlicense'
            )) NOT NULL,
            patentable INT NOT NULL,
            FOREIGN KEY (submitted) REFERENCES Users(username)
        );

        CREATE TABLE IF NOT EXISTS files (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            entry INTEGER NOT NULL,
            category TEXT CHECK ( category IN (
				'Poster',
				'Paper',
				'Presentation'
			)) NOT NULL,
            file BLOB NOT NULL,
            FOREIGN KEY (entry) REFERENCES Entries(id)
        );
    `)

	if err != nil {
		panic(err.Error())
	}
}

func renderTemplate(w http.ResponseWriter, r *http.Request, tplFile string, title string, data tplData) {
	if data == nil {
		data = tplData{
			"Title": title,
		}
	} else {
		data["Title"] = title
	}

	user := getUser(r)
	if user != "" {
		data["User"] = user
	}

	err := tpl.ExecuteTemplate(w, tplFile, data)
	if err != nil {
		debugPrint("Error Rendering Template", err)
		http.Error(w, "Error Rendering Template", http.StatusInternalServerError)
	}
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/img/favicon.ico")
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "about.html", "About", nil)
}

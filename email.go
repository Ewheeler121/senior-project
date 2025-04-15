package main

import (
	"database/sql"
	"math/rand"
	"net/http"
	"net/smtp"
	"os"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
var host = "smtp.gmail.com"
var port = "587"

func verify_email(email string) bool {
	return len(email) > 3 && len(email) < 252 && emailRegex.MatchString(email)
}

func send_email(target string, subject string, body string) error {
	from := os.Getenv("EMAIL")
	pwd := os.Getenv("EMAIL_PWD")
	to := []string{target}
	address := host + ":" + port
	msg := []byte(subject + "\r\n\r\n" + body)

	auth := smtp.PlainAuth("", from, pwd, host);
	
	return smtp.SendMail(address, auth, from, to, msg)
}

func forgotPWHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "forgotpw.html", "Forgot Password", nil)
}

func recoverPWHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpw/"):]})
}

func recoverPWPostHandler(w http.ResponseWriter, r *http.Request) {
	pwd := r.FormValue("password")
    hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	ver := r.URL.Path[len("/recoverpwpost/"):]
	
	if len(ver) <= 0 || ver == "" {
		renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpwpost/"):], "message": "some error occured"})
	}

    if err != nil {
		debugPrint("1:", err)
		renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpwpost/"):], "message": "some error occured hashing"})
        return
    }
	
	tx, err := db.Begin()
    if err != nil {
		debugPrint("2:", err)
		renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpwpost/"):], "message": "some error occured"})
        return
    }
	
	result, err := tx.Exec(`UPDATE users SET password = ? WHERE email_ver_hash = ?`, hash, ver)
	affected, _ := result.RowsAffected()
    if err != nil || affected == 0 {
		debugPrint("3:", err)
		renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpwpost/"):], "message": "some error occured"})
        return
    }

	result, err = tx.Exec(`UPDATE users SET email_ver_hash = NULL WHERE email_ver_hash = ?`, ver)
	affected, _ = result.RowsAffected()
    if err != nil || affected == 0 {
		debugPrint("4:", err)
		renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpwpost/"):], "message": "some error occured"})
        return
    }

	err = tx.Commit()
    if err != nil {
		debugPrint("5:", err)
		renderTemplate(w, r, "recoverpw.html", "Change Password", tplData{"ver": r.URL.Path[len("/recoverpwpost/"):], "message": "some error occured"})
        return
    }
	
	renderTemplate(w, r, "login.html", "Login", tplData{"message": "password changed"})
}

func forgotPWPostHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email");
	
	var username string
	err := db.QueryRow("SELECT username FROM users WHERE email = ?", username, email).Scan()
    if err != sql.ErrNoRows {
		debugPrint(err)
		renderTemplate(w, r, "forgotpw.html", "Forgot Password", tplData{ "message": "unknown email"})
		return
	}
	
	//TODO: CHECK IF MAGIC NUMBER IS IN DB ALREADY
	var alphaNumRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	emailVerRandRune := make([]rune, 64)
	for i := 0; i < 64; i++ {
		emailVerRandRune[i] = alphaNumRunes[rand.Intn(len(alphaNumRunes)-1)]
	}
	ver := string(emailVerRandRune)

	_, err = db.Exec(`UPDATE users SET email_ver_hash = ? WHERE email = ?`, ver, email)
	if err != nil {
		debugPrint(err)
		renderTemplate(w, r, "forgotpw.html", "Forgot Password", tplData{ "message": "error in hashing algorithm"})
		return
	}

	debugPrint(ver)
	err = send_email(email, "Reset Password", `recovery link: ` + os.Getenv("DOMAIN") + `/recoverpw/` + ver)
	if err != nil {
		debugPrint(err)
		renderTemplate(w, r, "forgotpw.html", "Forgot Password", tplData{ "message": "error, unable to send error"})
		return
	}
	renderTemplate(w, r, "forgotpw.html", "Forgot Password", tplData{ "message": "recovery email sent"})
}

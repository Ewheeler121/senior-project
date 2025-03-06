package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Poster struct {
	ID          int
	Title       string
	Submitted   string
	Author      string
	GradLevel   string
	Keywords    string
	Affiliation string
	Abstract    string
	Comments    string
	Category    string
	License     string
    Patentable  int
}

type File struct {
	entry    int
	category string
	file     []byte
}

func formatMultiString(input string) string {
	parts := strings.Split(input, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return strings.Join(parts, ",")
}

func submitPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "submit.html", "Submit", nil)
}

func submitPostHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)
	mu.Lock()
	defer mu.Unlock()

	var files []File
	upload := Poster {
		Title:       r.FormValue("title"),
		Submitted:   getUser(r),
		Author:      r.FormValue("authors"),
		GradLevel:   r.FormValue("gradlevel"),
		Keywords:    r.FormValue("keywords"),
		Affiliation: r.FormValue("affiliations"),
		Abstract:    r.FormValue("abstract"),
		Comments:    r.FormValue("comments"),
		Category:    r.FormValue("category"),
		License:     r.FormValue("license"),
	}
    if r.FormValue("patentable") != "" {
        upload.Patentable = 1;
    } else {
        upload.Patentable = 0;
    }

	for key, fileHeaders := range r.MultipartForm.File {
		if strings.HasPrefix(key, "file") {
			num := strings.TrimPrefix(key, "file")
			fileTypeKey := "filetype" + num
			fileType := r.FormValue(fileTypeKey)

			for _, fileHeader := range fileHeaders {
				file, err := fileHeader.Open()
				if err != nil {
					renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to Upload File"})
					return
				}
				defer file.Close()

				fileBytes, err := io.ReadAll(file)
				if err != nil {
					renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to Upload File"})
					return
				}

				// Scan file before saving
				clean, err := scanFile(fileBytes, fileHeader.Filename)
				if err != nil {
					renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Error scanning file for viruses"})
					fmt.Println("An error occured: ", err)
					return
				}
				if !clean {
					renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Virus detected! File upload rejected."})
					return
				}

				files = append(files, File {
					category: fileType,
					file:     fileBytes,
				})
			}
		}
	}

	//TODO: validate input here, current accepts everything and can crash when adding to database due to constrains

	//TODO: remove + err.Error() when done testing
	tx, err := db.Begin()
	if err != nil {
		renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to preform an SQL Query, database corrupt"})
		return
	}

	result, err := tx.Exec(`INSERT INTO entries (title, submitted, authors, gradlevel, affiliation, keywords, abstract, comments, category, license, patentable) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		upload.Title, upload.Submitted, upload.Author, upload.GradLevel, upload.Affiliation, upload.Keywords, upload.Abstract, upload.Comments, upload.Category, upload.License, upload.Patentable)
	if err != nil {
		tx.Rollback()
		renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to preform an SQL Query" + err.Error()})
		return
	}

	//TODO: could break if using different DB, use RETURNING if moving
	id, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to preform an SQL Query" + err.Error()})
		return
	}

	for _, file := range files {
		_, err := tx.Exec(`INSERT INTO files (entry, category, file) VALUES (?, ?, ?)`, id, file.category, file.file)
		if err != nil {
			tx.Rollback()
			renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to preform an SQL Query" + err.Error()})
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Unable to preform an SQL Query, database corrupt"})
		return
	}

	renderTemplate(w, r, "submit.html", "Submit", tplData{"message": "Poster Uploaded Successfully"})
}

func posterDownloadHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/download/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}

	var fileData []byte
	query := `SELECT file FROM files WHERE id = ?`
	err = db.QueryRow(query, id).Scan(&fileData)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Write(fileData)

}

func posterPageHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/poster/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}

	var poster Poster
	var files []int

	query := `SELECT title, submitted, authors, gradlevel, affiliation, keywords, abstract, comments, category, license, patentable FROM entries WHERE ID=?`
	err = db.QueryRow(query, id).Scan(&poster.Title, &poster.Submitted, &poster.Author, &poster.GradLevel, &poster.Affiliation, &poster.Keywords, &poster.Abstract, &poster.Comments, &poster.Category, &poster.License, &poster.Patentable)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`SELECT id FROM files WHERE entry=?`, id)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var e int
		if err := rows.Scan(&e); err != nil {
			http.Error(w, "Error Searching Database", http.StatusBadRequest)
			return
		}
		files = append(files, e)
	}

	renderTemplate(w, r, "poster.html", "Submit", tplData{"poster": poster, "files": files})
}

func searchPageHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "search.html", "Search", nil)
}

func searchPostHandler(w http.ResponseWriter, r *http.Request) {
	baseQuery := `SELECT id, title, authors, affiliation, keywords, category, license FROM entries WHERE 1=1`
	var queryParams []interface{}
	if r.FormValue("title") != "" {
		baseQuery += " AND title LIKE ?"
		queryParams = append(queryParams, "%"+r.FormValue("title")+"%")
	}

	if r.FormValue("category") != "" {
		baseQuery += " AND category LIKE ?"
		queryParams = append(queryParams, "%"+r.FormValue("category")+"%")
	}

	if r.FormValue("author") != "" {
		for _, a := range strings.Split(r.FormValue("author"), ",") {
			baseQuery += " AND authors LIKE ?"
			queryParams = append(queryParams, "%"+a+"%")
		}
	}

	if r.FormValue("keyword") != "" {
		for _, k := range strings.Split(r.FormValue("keyword"), ",") {
			baseQuery += " AND keywords LIKE ?"
			queryParams = append(queryParams, "%"+k+"%")
		}
	}

	if r.FormValue("affiliation") != "" {
		for _, a := range strings.Split(r.FormValue("affiliation"), ",") {
			baseQuery += " AND affiliation LIKE ?"
			queryParams = append(queryParams, "%"+a+"%")
		}
	}

	rows, err := db.Query(baseQuery, queryParams...)
	if err != nil {
		fmt.Println("1" + err.Error())
		http.Error(w, "Error Searching Database", http.StatusBadRequest)
		return
	}
	defer rows.Close()

	var results []Poster
	for rows.Next() {
		var p Poster
		if err := rows.Scan(&p.ID, &p.Title, &p.Author, &p.Affiliation, &p.Keywords, &p.Category, &p.License); err != nil {
			fmt.Println("2" + err.Error())
			http.Error(w, "Error Searching Database", http.StatusBadRequest)
			return
		}
		results = append(results, p)
	}

	renderTemplate(w, r, "search.html", "Search", tplData{"results": results})
}

func deleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/delete/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}

	var submitted string
	query := `SELECT submitted FROM entries WHERE id = ?`
	err = db.QueryRow(query, id).Scan(&submitted)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	if getUser(r) != submitted {
		http.Error(w, "Permission Denied", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`DELETE FROM entries WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Error deleting entry", http.StatusBadRequest)
		return
	}

	//TODO: change back to profile instead of search
	http.Redirect(w, r, "/search", 302)
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/deleteFile/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}

	var entry int
	query := `SELECT entry FROM files WHERE id = ?`
	err = db.QueryRow(query, id).Scan(&entry)
	if err != nil {
		http.Error(w, "Invalid entry ID", http.StatusBadRequest)
		return
	}

	var submitted string
	query = `SELECT submitted FROM entries WHERE id = ?`
	err = db.QueryRow(query, entry).Scan(&submitted)
	if err != nil {
		http.Error(w, "Invalid paper ID", http.StatusBadRequest)
		return
	}

	if getUser(r) != submitted {
		http.Error(w, "Permission Denied", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`DELETE FROM files WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Error deleting file", http.StatusBadRequest)
		return
	}

	//TODO: ie there are no more files then we might delete the poster???
	http.Redirect(w, r, "/poster/"+strconv.Itoa(id), 302)
}

package main

import (
	"strings"
)

type Entry struct {
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

func getEntryById(id int) (Entry, error) {
	var entry Entry 
	query := `SELECT id, title, submitted, authors, gradlevel, affiliation, keywords, abstract, comments, category, license, patentable FROM entries WHERE ID=?`
	err := db.QueryRow(query, id).Scan(&entry.ID, &entry.Title, &entry.Submitted, &entry.Author, &entry.GradLevel, &entry.Affiliation, &entry.Keywords, &entry.Abstract, &entry.Comments, &entry.Category, &entry.License, &entry.Patentable)
	if err != nil {
		debugPrint("Error getting entry", err)
		return entry, err
	}
	return entry, nil
}

func formatMultiInput(input string) string {
	parts := strings.Split(input, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	return strings.Join(parts, ", ")
}

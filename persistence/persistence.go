package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/temelpa/timetravel/entity"
	"log"
	"time"
)

// We are passing db reference connection from main to our method with other parameters
func addRecordVersion(id int, body map[string]*string) {
	//db *sql.DB,
	db, _ := sql.Open("sqlite3", "./records.db") // Open the created SQLite File
	defer db.Close()                             // Defer closing the database
	log.Println("Inserting student record ...")

	now := time.Now()
	updateMap := map[string]string{}
	record, _ := getLatestVersionForRecord(id)
	//New record
	if record.ID == -1 {
		insertRecordSQL := `INSERT INTO record(id, createdAt, updatedAt) VALUES (?, ?, ?)`
		statement, err := db.Prepare(insertRecordSQL) // Prepare statement.
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec(id, now.Unix(), now.Unix())
		if err != nil {
			log.Fatalln(err.Error())
		}
	}
	insertRecordVersionSQL := `INSERT INTO recordVersion(id) VALUES (?, ?)`
	statement, err := db.Prepare(insertRecordVersionSQL) // Prepare statement.
	if err != nil {
		log.Fatalln(err.Error())
	}
	res, err := statement.Exec(id)
	if err != nil {
		log.Fatalln(err.Error())
	}
	versionId, err := res.LastInsertId()
	if err != nil {
		log.Fatalln(err.Error())
	}
	record.VersionID = versionId

	//Get existing values of last version that are not part of current update request
	for key, value := range record.Data {
		if _, ok := body[key]; !ok {
			updateMap[key] = value
		}
	}
	for key, value := range body {
		if _, ok := updateMap[key]; ok && value == nil {
			delete(updateMap, key)
		} else if value != nil {
			updateMap[key] = *value
		}
	}

	for key, value := range updateMap {
		insertRecordVersionFieldSQL := `INSERT INTO recordVersionField(versionId, key, value) VALUES (?, ?, ?)`
		statement, err := db.Prepare(insertRecordVersionFieldSQL) // Prepare statement.
		if err != nil {
			log.Fatalln(err.Error())
		}
		_, err = statement.Exec(record.VersionID, key, value)
		if err != nil {
			log.Fatalln(err.Error())
		}
	}

}

func getLatestVersionForRecord(id int) (entity.Record, error) {
	//db *sql.DB

	db, _ := sql.Open("sqlite3", "./records.db") // Open the created SQLite File
	defer db.Close()                             // Defer closing the database
	var versionId int64
	var props map[string]string
	var record = entity.Record{}
	record.ID = -1

	if err := db.QueryRow("SELECT versionId FROM recordVersion where recordId = ? "+
		"ORDER BY versionId DESC LIMIT 1", id).Scan(&versionId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return record, fmt.Errorf("Record %d: not found", id)
		}
		return record, fmt.Errorf("Record %d: %v", id, err)
	}
	log.Println("Version Id: ", versionId, " Record Id: ", id)
	record.ID = id
	record.VersionID = versionId
	rows, err := db.Query("SELECT key, value FROM recordVersionField WHERE versionId = ?", versionId)
	if err != nil {
		return record, err
	}
	defer rows.Close()

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			record.Data = props
			return record, err
		}
		props[key] = value
	}
	record.Data = props
	if err = rows.Err(); err != nil {
		return record, err
	}
	return record, nil
}

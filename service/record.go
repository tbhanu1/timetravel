package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/temelpa/timetravel/entity"
	//"github.com/temelpa/timetravel/persistence"
	_ "github.com/mattn/go-sqlite3"
)

var ErrRecordDoesNotExist = errors.New("record with that id does not exist")
var ErrRecordIDInvalid = errors.New("record id must >= 0")
var ErrRecordAlreadyExists = errors.New("record already exists")

// Implements method to get, create, and update record data.
type RecordService interface {

	// GetRecord will retrieve an record.
	GetRecord(ctx context.Context, id int) (entity.Record, error)

	// GetVersionedRecord will retrieve an record with a specific version.
	GetVersionedRecord(ctx context.Context, id int, versionId int64) (entity.Record, error)

	GetVersionIdsForRecord(ctx context.Context, id int) ([]int64, error)

	// CreateRecord will insert a new record.
	//
	// If it a record with that id already exists it will fail.
	CreateRecord(ctx context.Context, record entity.Record) error

	// UpdateRecord will change the internal `Map` values of the record if they exist.
	// if the update[key] is null it will delete that key from the record's Map.
	//
	// UpdateRecord will error if id <= 0 or the record does not exist with that id.
	UpdateRecord(ctx context.Context, id int, updates map[string]*string) (entity.Record, error)
}

// InMemoryRecordService is an in-memory implementation of RecordService.
type InMemoryRecordService struct {
	data map[int]entity.Record
}

type RepositoryRecordService struct {
	db *sql.DB
}

func NewInMemoryRecordService() InMemoryRecordService {
	return InMemoryRecordService{
		data: map[int]entity.Record{},
	}
}

func NewRepositoryRecordService(sqldb *sql.DB) RepositoryRecordService {
	return RepositoryRecordService{
		sqldb,
	}
}

func (s *InMemoryRecordService) GetRecord(ctx context.Context, id int) (entity.Record, error) {
	record := s.data[id]
	if record.ID == 0 {
		return entity.Record{}, ErrRecordDoesNotExist
	}

	record = record.Copy() // copy is necessary so modifations to the record don't change the stored record
	return record, nil
}

func (s *InMemoryRecordService) GetVersionedRecord(ctx context.Context, id int, versionId int64) (entity.Record, error) {
	return entity.Record{}, nil
}

func (s *InMemoryRecordService) GetVersionIdsForRecord(ctx context.Context, id int) ([]int64, error) {
	return nil, nil
}

func (s *InMemoryRecordService) CreateRecord(ctx context.Context, record entity.Record) error {
	id := record.ID
	if id <= 0 {
		return ErrRecordIDInvalid
	}

	existingRecord := s.data[id]
	if existingRecord.ID != 0 {
		return ErrRecordAlreadyExists
	}

	s.data[id] = record
	return nil
}

func (s *InMemoryRecordService) UpdateRecord(ctx context.Context, id int, updates map[string]*string) (entity.Record, error) {
	entry := s.data[id]
	if entry.ID == 0 {
		return entity.Record{}, ErrRecordDoesNotExist
	}

	for key, value := range updates {
		if value == nil { // deletion update
			delete(entry.Data, key)
		} else {
			entry.Data[key] = *value
		}
	}

	return entry.Copy(), nil
}

func (s *RepositoryRecordService) GetRecord(ctx context.Context, id int) (entity.Record, error) {
	return getLatestVersionForRecord(s.db, id)
}

func (s *RepositoryRecordService) GetVersionIdsForRecord(ctx context.Context, id int) ([]int64, error) {
	return getVersionIdsForRecord(s.db, id)
}

func (s *RepositoryRecordService) GetVersionedRecord(ctx context.Context, id int, versionId int64) (entity.Record, error) {
	return getSpecifiedVersionForRecord(s.db, id, versionId)
}

func (s *RepositoryRecordService) CreateRecord(ctx context.Context, record entity.Record) error {
	//addRecordVersion(record.ID, record.Data)
	return nil
}

func (s *RepositoryRecordService) UpdateRecord(ctx context.Context, id int, updates map[string]*string) (entity.Record, error) {
	addRecordVersion(s.db, id, updates)
	return getLatestVersionForRecord(s.db, id)
}

func getLatestVersionForRecord(db *sql.DB, id int) (entity.Record, error) {

	var versionId int64
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
	rows, err := db.Query("SELECT key, value FROM recordVersionField WHERE fieldVersionId = ?", versionId)
	if err != nil {
		return record, err
	}
	defer rows.Close()
	var props = map[string]string{}
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

func addRecordVersion(db *sql.DB, id int, body map[string]*string) {
	log.Println("Inserting record version and field values ...")

	now := time.Now()
	updateMap := map[string]string{}
	record, _ := getLatestVersionForRecord(db, id)
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
	insertRecordVersionSQL := `INSERT INTO recordVersion(recordId) VALUES (?)`
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
	record.ID = id
	record.VersionID = versionId

	//Get existing values of last version that are not part of current update request
	for key, value := range record.Data {
		if _, ok := body[key]; !ok {
			updateMap[key] = value
		}
	}
	for key, value := range body {
		//Skip keys with null values
		//If it existed in the last version, it has not been populated either.
		if value != nil {
			updateMap[key] = *value
		}
	}

	for key, value := range updateMap {
		insertRecordVersionFieldSQL := `INSERT INTO recordVersionField(fieldVersionId, key, value) VALUES (?, ?, ?)`
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

func getSpecifiedVersionForRecord(db *sql.DB, id int, versionId int64) (entity.Record, error) {
	var record = entity.Record{}
	record.ID = -1

	record.ID = id
	record.VersionID = versionId
	rows, err := db.Query("SELECT key, value FROM recordVersionField WHERE fieldVersionId = ?", versionId)
	if err != nil {
		return record, err
	}
	defer rows.Close()
	var props = map[string]string{}
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

func getVersionIdsForRecord(db *sql.DB, id int) ([]int64, error) {
	rows, err := db.Query("SELECT versionId FROM recordVersion WHERE recordId = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	versionIds := []int64{}
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var verId int64
		if err := rows.Scan(&verId); err != nil {

			return versionIds, err
		}
		versionIds = append(versionIds, verId)
	}
	if err = rows.Err(); err != nil {
		return versionIds, err
	}
	return versionIds, nil
}

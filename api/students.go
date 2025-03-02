package api

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type Student struct {
	StudentID        int    `json:"studentId"`
	ParentID         int    `json:"parentId"`
	FirstName        string `json:"firstName"`
	LastName         string `json:"lastName"`
	Grade           string `json:"grade"`
	AdmNumber       string `json:"admNumber"`
	Address         string `json:"address"`
	EmergencyContact string `json:"emergencyContact"`
	IsActive        bool   `json:"isActive"`
}

// HandleAddStudent handles adding a single student
func HandleAddStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var student Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		result, err := db.Exec(`
			INSERT INTO Students (ParentID, FirstName, LastName, Grade, AdmNumber, 
				PickupPoint, DropoffPoint, EmergencyContact, IsActive)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, true)`,
			student.ParentID, student.FirstName, student.LastName, student.Grade,
			student.AdmNumber, student.Address, student.EmergencyContact)
		
		if err != nil {
			log.Printf("Error adding student: %v", err)
			http.Error(w, "Error adding student", http.StatusInternalServerError)
			return
		}

		id, _ := result.LastInsertId()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Student added successfully",
			"studentId": id,
		})
	}
}

// HandleBulkUploadStudents handles bulk upload of students via CSV
func HandleBulkUploadStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Error receiving file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		// Skip header row
		_, err = reader.Read()
		if err != nil {
			http.Error(w, "Error reading CSV header", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		stmt, err := tx.Prepare(`
			INSERT INTO Students (ParentID, FirstName, LastName, Grade, AdmNumber, 
				PickupPoint, DropoffPoint, EmergencyContact, IsActive)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, true)`)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		var success, failed int
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			parentID, _ := strconv.Atoi(record[0])
			_, err = stmt.Exec(
				parentID,
				record[1], // FirstName
				record[2], // LastName
				record[3], // Grade
				record[4], // AdmNumber
				record[5], // Address
				record[6], // EmergencyContact
			)

			if err != nil {
				failed++
				continue
			}
			success++
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			http.Error(w, "Error committing transaction", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Uploaded %d students successfully, %d failed", success, failed),
		})
	}
}

// HandleUpdateStudent handles updating student information
func HandleUpdateStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var student Student
		if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		_, err := db.Exec(`
			UPDATE Students 
			SET FirstName = ?, LastName = ?, Grade = ?, AdmNumber = ?,
				Address = ?, EmergencyContact = ?
			WHERE StudentID = ?`,
			student.FirstName, student.LastName, student.Grade, student.AdmNumber,
			student.Address, student.EmergencyContact,
			student.StudentID)

		if err != nil {
			log.Printf("Error updating student: %v", err)
			http.Error(w, "Error updating student", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Student updated successfully",
		})
	}
}

// HandleDeleteStudent handles deleting a student (soft delete)
func HandleDeleteStudent(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		studentID := r.URL.Query().Get("id")
		if studentID == "" {
			http.Error(w, "Student ID is required", http.StatusBadRequest)
			return
		}

		_, err := db.Exec("UPDATE Students SET IsActive = false WHERE StudentID = ?", studentID)
		if err != nil {
			log.Printf("Error deleting student: %v", err)
			http.Error(w, "Error deleting student", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Student deleted successfully",
		})
	}
}

// HandleListStudents handles retrieving all active students
func HandleListStudents(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`
			SELECT StudentID, ParentID, FirstName, LastName, Grade, AdmNumber,
				   Address, EmergencyContact
			FROM Students
			WHERE IsActive = true
			ORDER BY Grade, LastName, FirstName`)
		if err != nil {
			log.Printf("Error getting students: %v", err)
			http.Error(w, "Error getting students list", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var students []Student
		for rows.Next() {
			var s Student
			err := rows.Scan(&s.StudentID, &s.ParentID, &s.FirstName, &s.LastName,
				&s.Grade, &s.AdmNumber, &s.Address, &s.EmergencyContact)
			if err != nil {
				continue
			}
			students = append(students, s)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"students": students,
		})
	}
}

package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type statusResponse struct {
	Alive       bool     `json:"alive"`
	Application string   `json:"application"`
	Email       string   `json:"email"`
	Version     string   `json:"version"`
	Errors      []string `json:"errors"`
	Environment string   `json:"environment"`
}

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ResponseMessage struct {
	Message string `json:"message"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("postgres", "user=postgres password=selasie_123 dbname=users sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	router := gin.Default()

	// status check endpoint
	router.GET("/", statusCheck)

	// CRUD endpoints
	router.POST("/users", createUser)
	router.GET("/users", getAllUsers)
	router.GET("/users/:id", getUser)
	router.PUT("/users/:id", updateUser)
	router.DELETE("/users/:id", deleteUser)

	log.Println("Server starting on :8080")
	err = router.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}

func statusCheck(c *gin.Context) {
	var alive bool
	var errors []string

	// Check database connection
	err := db.Ping()
	if err != nil {
		alive = false
		errors = append(errors, "Database connection failed")
	} else {
		alive = true
	}

	response := statusResponse{
		Alive:       alive,
		Application: "My Management System",
		Email:       "jkpodo05@gmail.com",
		Version:     "1.0.0",
		Errors:      errors,
		Environment: "staging",
	}

	c.JSON(http.StatusOK, response)
}

func createUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Check if the user already exists
	var existingUserID int
	err := db.QueryRow("SELECT id FROM users WHERE name=$1 OR email=$2", user.Name, user.Email).Scan(&existingUserID)
	if err != nil && err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	if existingUserID != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this name or email already exists"})
		return
	}

	// Insert the user into the database
	var lastInsertID int
	err = db.QueryRow("INSERT INTO users(name, email) VALUES($1, $2) returning id;", user.Name, user.Email).Scan(&lastInsertID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "User created successfully", "user_id": lastInsertID})
}

func getAllUsers(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var name, email string
		err := rows.Scan(&id, &name, &email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		user := map[string]interface{}{
			"id":    id,
			"name":  name,
			"email": email,
		}
		users = append(users, user)
	}

	err = rows.Err()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	var name, email string

	err := db.QueryRow("SELECT id, name, email FROM users WHERE id = $1", id).Scan(&id, &name, &email)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user := map[string]interface{}{
		"id":    id,
		"name":  name,
		"email": email,
	}

	c.JSON(http.StatusOK, user)
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec("UPDATE users SET name=$1, email=$2 WHERE id=$3", user.Name, user.Email, id)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"message": "User with this name or email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, ResponseMessage{Message: "User updated successfully"})
}

func deleteUser(c *gin.Context) {
	id := c.Param("id")

	result, err := db.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		return
	}

	c.JSON(http.StatusOK, ResponseMessage{Message: "User deleted successfully"})
}

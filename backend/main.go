package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/couchbase/gocb/v2"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var cluster *gocb.Cluster
var bucket *gocb.Bucket
var collection *gocb.Collection

func initCouchbase() {
	var err error
	cluster, err = gocb.Connect("couchbase://couchbase", gocb.ClusterOptions{
		Username: "admin",
		Password: "password",
	})
	if err != nil {
		log.Fatalf("Could not connect to Couchbase: %v", err)
	}

	bucket = cluster.Bucket("mybucket")

	// Wait for the bucket to be ready
	for i := 0; i < 30; i++ {
		err = bucket.WaitUntilReady(5*time.Second, nil)
		if err == nil {
			break
		}
		fmt.Println("Waiting for bucket to be ready...")
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Bucket is not ready: %v", err)
	}

	collection = bucket.DefaultCollection()
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = collection.Insert(user.Username, user, nil)
	if err != nil {
		http.Error(w, "Could not register user", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Registration successful"))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	getResult, err := collection.Get(user.Username, nil)
	if err != nil {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	var storedUser User
	err = getResult.Content(&storedUser)
	if err != nil {
		http.Error(w, "Could not retrieve user", http.StatusInternalServerError)
		return
	}

	if storedUser.Password != user.Password {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	w.Write([]byte("Login successful"))
}

func main() {
	initCouchbase()

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

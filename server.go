package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// User represents a user document in MongoDB
type User struct {
	ID         primitive.ObjectID	`bson:"_id" json:"id"`
	Username   string               `bson:"username" json:"username"`
	Password   string               `bson:"password" json:"password"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}


// ErrorResponse for consistent error handling
type ErrorResponse struct {
	Error string `json:"error"`
}

var client *mongo.Client
var collection *mongo.Collection


func main(){
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().ApplyURI("mongodb://admin:admin@localhost:27017/e-comm-dev?authSource=admin")
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil{
		log.Fatal("Failed to connect mongodb:", err)
	}
	defer client.Disconnect(ctx)	

	//ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}
	fmt.Println("Connected to MongoDB")

	// Set collection (using 'testdb' database and 'users' collection)
	collection = client.Database("Gotestdb").Collection("users")

	//Initialise router
	r := mux.NewRouter()
	r.Use(loggingMiddleware)

	//Define endpoints
	r.HandleFunc("/users", createUserHandler).Methods("POST")
	r.HandleFunc("/users/{id}", getUserHandler ).Methods("GET")
	r.HandleFunc("/users", getAllUsersHandler).Methods("GET")
	// r.HandleFunc("/users/{id}", updateUserHandler).Methods("PUT")
	// r.HandleFunc("/users/{id}", deleteUserHandler).Methods("DELETE")

	//start server 
	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

//create user
func createUserHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "application/json")
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		sendError(w, "invalid request payload", http.StatusBadRequest)
		return 
	}
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		sendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		sendError(w, "Error Encoding JSON", http.StatusInternalServerError)
	}
}

//get user
func getUserHandler(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	id, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		sendError(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	
	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err == mongo.ErrNoDocuments{
		sendError(w, "user not found", http.StatusNotFound)
		return
	}else if err != nil {
		sendError(w, "Failed to fetch user", http.StatusInternalServerError)
	}

	if err:= json.NewEncoder(w).Encode(user); err != nil {
		sendError(w, "Error encoding JSON", http.StatusInternalServerError)
	}
}

//update the user 
func getAllUsersHandler(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		sendError(w, "Failed to fetch Users", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	var users  []User
	if err = cursor.All(ctx, &users); err != nil {
		sendError(w, "Failed to decode users", http.StatusInternalServerError)
		return
	}

	if err:= json.NewEncoder(w).Encode(users); err != nil {
		sendError(w, "Error encoding JSON", http.StatusInternalServerError)
		return
	}
}

//delete the user
func sendError(w http.ResponseWriter, message string, status int){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil{
		log.Printf("Error encoding error response: %v", err)
	}
}

func loggingMiddleware(next http.Handler) http.Handler  {
	return  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
	
}
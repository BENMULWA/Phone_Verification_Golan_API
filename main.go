// PHONE VERIFICATION API BACKEND FOR TELCOME INTERGRATION
// This api validates and verifies phone numbers in correct format that is: it shld a kenya phone number

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"          // it encodes and decodes BSON data (Binary JSON) used by MongoDB
	"go.mongodb.org/mongo-driver/mongo"         // it interacts with mongoDB databases and collections in Go that lets it to connect to MongoDB, perform CRUD operations, and manage data
	"go.mongodb.org/mongo-driver/mongo/options" // it provides options for configuring MongoDB operations
)

// DATA STRUCTURES with OTP CODE VERIFICATION
// PhoneNumberRequest represents the request structure for phone number validation

type PhoneNumberRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// adding OTP code verification

type OTPVerifyRequest struct {
	Phone string `json:"phone_number"`
	OTP   string `json:"otp"`
}

type PhoneNumberResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// MongoDB connection settings

var collection *mongo.Collection

// Connect to MongoDB and get the collection handle
func connectMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().ApplyURI(
		"mongodb+srv://mulwabenard9507:benard9507@cluster0.xad7ngd.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0").SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal("❌ MongoDB connection error:", err)
	}

	// Ping the DB to confirm connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("❌ MongoDB ping failed:", err)
	}

	// Connect to database `otp_api`, and access a collection `verifications`
	collection = client.Database("otpapi").Collection("verifications")

	log.Println("✅ Connected to MongoDB and got 'verifications' collection")
}

// Dummy handler to show it works
func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "MongoDB connected and 'verifications' collection ready.")
}

// Generate a 6-digit OTP

func generateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(900000)+100000)
}

// PHONE NUMBER VALIDATION FUNCTION
func isvalidPhoneNumber(phonenumber string) (bool, string) { // takes a phone number as input and removes all spaces from it
	phonenumber = strings.ReplaceAll(phonenumber, " ", "")
	pattern := `^(\+254|0)(7\d{8}|11\d{7})$` // Regular expression pattern for Kenyan phone numbers
	matched, _ := regexp.MatchString(pattern, phonenumber)

	if matched {
		return true, "Valid Phone Number"
	} else {
		return false, "Invalid Phone Number"
	}
}

// HHTP HANDLER FUNCTION
func verifyPhoneNumberhandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid Phone Number Format", http.StatusBadRequest)
		return
	}

	var request PhoneNumberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	otp := generateOTP()
	expiry := time.Now().Add(5 * time.Minute) // Set OTP expiry time to 5 minutes

	_, err := collection.InsertOne(r.Context(), bson.M{
		"phone":      request.PhoneNumber,
		"otp":        otp,
		"expires_at": expiry,
		"verified":   false,
	})

	if err != nil {
		http.Error(w, "Failed to save phone number to database", http.StatusInternalServerError)
		return
	}

	// simulate SMS (print to console)
	fmt.Printf("OTP for %s is: %s (expires at %s)\n", request.PhoneNumber, otp, expiry.Format(time.RFC3339))

	json.NewEncoder(w).Encode(map[string]string{
		"message": "OTP sent successfully",
	})

	isValid, message := isvalidPhoneNumber(request.PhoneNumber)
	response := PhoneNumberResponse{
		Valid:   isValid,
		Message: message,
	}

	json.NewEncoder(w).Encode(response)
}

// endpoint to verify the OTP
func verifyOTP(w http.ResponseWriter, r *http.Request) {
	var req OTPVerifyRequest
	json.NewDecoder(r.Body).Decode(&req)

	filter := bson.M{
		"phone":      req.Phone,
		"otp":        req.OTP,
		"expires_at": bson.M{"$gt": time.Now()}, // Check if OTP has not expired
	}
	var result bson.M
	err := collection.FindOne(r.Context(), filter).Decode(&result)
	if err != nil {
		http.Error(w, "Invalid OTP or It has expired", http.StatusUnauthorized)
		return
	}

	// mark the OTP as verified
	collection.UpdateOne(context.TODO(), filter, bson.M{"$set": bson.M{"verified": true}})

	json.NewEncoder(w).Encode(map[string]string{
		"message": "OTP verified successfully and its still working",
	})
}

// MAIN FUNCTION
// MAIN FUNCTION
func main() {
	connectMongoDB() // ✅ Call this before anything else

	http.HandleFunc("/V1/verify-phone_number", verifyPhoneNumberhandler)
	http.HandleFunc("/V1/request-otp", verifyPhoneNumberhandler) // Same handler for request
	http.HandleFunc("/V1/verify-otp", verifyOTP)

	fmt.Println("🚀 Server running at http://localhost:8000")
	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal("❌ Error starting server:", err)
	}
}

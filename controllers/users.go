package controllers

import (
	"college-bazar-backend/models"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type JWTPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	jwt.RegisteredClaims
}

type UserService struct {
	logger          *zap.Logger
	usersCollection *mongo.Collection
}

func NewUserService(logger *zap.Logger, usersCollection *mongo.Collection) *UserService {
	return &UserService{
		logger:          logger,
		usersCollection: usersCollection,
	}
}

func (us *UserService) Register(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Image    string `json:"image"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingUser models.User
	err := us.usersCollection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		http.Error(w, "User already exists", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	newUser := &models.User{
		ID:       primitive.NewObjectID(),
		Username: req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
		Image:    req.Image,
		Cart:     []primitive.ObjectID{},
	}

	_, err = us.usersCollection.InsertOne(ctx, newUser)
	if err != nil {
		http.Error(w, "Error saving user", http.StatusInternalServerError)
		return
	}

	token, err := generateJWT(newUser.ID.Hex(), newUser.Username)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully", "token": token})
}

func (us *UserService) Login(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := us.usersCollection.FindOne(ctx, bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid email address", http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid password", http.StatusBadRequest)
		return
	}

	token, err := generateJWT(user.ID.Hex(), user.Username)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "User logged in successfully", "token": token})
}

func (us *UserService) GetCartItems(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(models.UserIDKey)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := us.usersCollection.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user.Cart)
}

func (us *UserService) AddProductToCart(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(models.UserIDKey)
	var req struct {
		ProductID string `json:"productId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := us.usersCollection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$push": bson.M{"cart": req.ProductID}})
	if err != nil {
		http.Error(w, "Error adding product to cart", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Product added to cart successfully"})
}

func (us *UserService) RemoveProductFromCart(w http.ResponseWriter, r *http.Request) {

	userID := r.Context().Value(models.UserIDKey)
	var req struct {
		ProductID string `json:"productId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := us.usersCollection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$pull": bson.M{"cart": req.ProductID}})
	if err != nil {
		http.Error(w, "Error removing product from cart", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Product removed from cart successfully"})
}

func generateJWT(id, name string) (string, error) {

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "some-random-jwt-secret"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, JWTPayload{
		ID:   id,
		Name: name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})

	return token.SignedString([]byte(jwtSecret))
}

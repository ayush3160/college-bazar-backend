package main

import (
	models "college-bazar-backend/models"
	httpServer "college-bazar-backend/routes"
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const defaultPort = "8081"

func setupLogger() *zap.Logger {
	logCfg := zap.NewDevelopmentConfig()

	// Customize the encoder config to put the emoji at the beginning.
	logCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	if *models.IsDebugLevel {
		logCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		logCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
		logCfg.EncoderConfig.EncodeCaller = nil
	}

	logCfg.DisableStacktrace = true
	logger, err := logCfg.Build()
	if err != nil {
		log.Panic("failed to start the logger for the server")
		return nil
	}
	return logger
}

func flagInit() {
	models.IsDebugLevel = flag.Bool("debug", false, "Enable debug level logs")
	models.IsDevelopment = flag.Bool("development", false, "Enable development mode")

	flag.Parse()
}

func main() {

	flagInit()
	logger := setupLogger()

	err := godotenv.Load(".env")
	if err != nil {
		if *models.IsDevelopment {
			err = godotenv.Load(".env.local")
			if err != nil {
				logger.Warn("Error loading .env.local file")
			}
		} else {
			logger.Warn("Error loading .env file proceeding with default values")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	mongoURI := os.Getenv("MONGO_URI")

	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	clientOptions := options.Client().
		ApplyURI(mongoURI).
		SetRetryWrites(true).
		SetRetryReads(true)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Error connecting to the database", zap.Error(err))
		return
	}

	if err := client.Ping(ctx, nil); err != nil {
		logger.Error("Error reaching to database", zap.Error(err))
		return
	}

	logger.Info("Connected to the database")

	dbName := os.Getenv("MONGO_DB_NAME")

	logger.Info("Database Name", zap.String("name", dbName))

	if dbName != "" {
		dbName = "event-trigger"
	}

	mongoDb := client.Database(dbName)

	// Create Collections
	usersCollection := mongoDb.Collection("users")
	productsCollection := mongoDb.Collection("products")

	serverCtx := context.Background()

	r := chi.NewRouter()

	httpServer.New(r, logger, usersCollection, productsCollection)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
		BaseContext: func(listener net.Listener) context.Context {
			return serverCtx
		},
	}

	logger.Info("Starting the Server", zap.String("port", port))

	go func() {

		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

		<-quit
		log.Println("Shutting down the server...")

		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v", err)
		}

		log.Println("Server stopped")
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Could not start the server", zap.Error(err))
	}
}

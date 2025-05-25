package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"

    firebase "firebase.google.com/go"
    "firebase.google.com/go/auth"
    _ "github.com/lib/pq"
    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "google.golang.org/api/option"
)

var (
    db *sql.DB
    firebaseApp *firebase.App
    firebaseAuthClient *auth.Client
)

type RegisterRequest struct {
    PhoneNumber string `json:"phone_number"`
    Name        string `json:"name"`
    Email       string `json:"email"`
}

func initFirebase() error {
    ctx := context.Background()
    sa := option.WithCredentialsFile(os.Getenv("FIREBASE_CREDENTIALS_FILE"))
    app, err := firebase.NewApp(ctx, nil, sa)
    if err != nil {
        return err
    }
    firebaseApp = app

    client, err := app.Auth(ctx)
    if err != nil {
        return err
    }
    firebaseAuthClient = client
    return nil
}

func setUserID(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, "user_id", userID)
}

func getUserID(ctx context.Context) string {
    if userID, ok := ctx.Value("user_id").(string); ok {
        return userID
    }
    return ""
}

func main() {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }

    // Initialize database connection
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        dbURL = "postgres://postgres:postgres@localhost:5432/milkpro_mlm?sslmode=disable"
    }

    db, err = sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatalf("Error opening database: %v", err)
    }
    defer db.Close()

    // Test database connection
    if err = db.Ping(); err != nil {
        log.Fatalf("Error connecting to database: %v", err)
    }
    log.Println("Successfully connected to database")

    r := mux.NewRouter()

    // Skip Firebase initialization for development
    log.Println("Warning: Running without Firebase authentication")

    // Health check endpoint
    r.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": "ok",
            "timestamp": time.Now().Format(time.RFC3339),
            "version": "1.0.0",
        })
    }).Methods("GET")

    // Products endpoint
    r.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
        rows, err := db.Query("SELECT id, name, type, price FROM products")
        if err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Failed to fetch products",
            })
            return
        }
        defer rows.Close()

        var products []map[string]interface{}
        for rows.Next() {
            var id int
            var name, productType string
            var price float64
            
            if err := rows.Scan(&id, &name, &productType, &price); err != nil {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusInternalServerError)
                json.NewEncoder(w).Encode(map[string]string{
                    "error": "Failed to parse product data",
                })
                return
            }

            products = append(products, map[string]interface{}{
                "id":    id,
                "name":  name,
                "type":  productType,
                "price": price,
            })
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "products": products,
            "total":    len(products),
        })
    }).Methods("GET")

    // User registration endpoint
    r.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
        var req RegisterRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Invalid request body",
            })
            return
        }

        // Validate required fields
        if req.PhoneNumber == "" || req.Name == "" {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusBadRequest)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Phone number and name are required",
            })
            return
        }

        // Check if user already exists
        var exists bool
        err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE phone = $1)", req.PhoneNumber).Scan(&exists)
        if err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Failed to check user existence",
            })
            return
        }

        if exists {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusConflict)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "User already exists",
            })
            return
        }

        // Insert user into database
        var userID int
        err = db.QueryRow(
            "INSERT INTO users (phone, name, email) VALUES ($1, $2, $3) RETURNING id",
            req.PhoneNumber, req.Name, req.Email,
        ).Scan(&userID)

        if err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Failed to create user",
            })
            return
        }

        // Fetch the created user
        var user struct {
            ID        int       `json:"id"`
            Phone     string    `json:"phone"`
            Name      string    `json:"name"`
            Email     string    `json:"email"`
            CreatedAt time.Time `json:"created_at"`
        }

        err = db.QueryRow(
            "SELECT id, phone, name, email, created_at FROM users WHERE id = $1",
            userID,
        ).Scan(&user.ID, &user.Phone, &user.Name, &user.Email, &user.CreatedAt)

        if err != nil {
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusInternalServerError)
            json.NewEncoder(w).Encode(map[string]string{
                "error": "Failed to fetch created user",
            })
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "message": "User registered successfully",
            "user":    user,
        })
    }).Methods("POST")

    fmt.Println("Starting server on :8081")
    log.Fatal(http.ListenAndServe(":8081", r))
}

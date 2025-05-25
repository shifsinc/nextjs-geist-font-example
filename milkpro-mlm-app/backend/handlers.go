package main

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "os"
    "strings"

    firebase "firebase.google.com/go"
    "firebase.google.com/go/auth"
    "golang.org/x/net/context"
    "google.golang.org/api/option"
)

var firebaseApp *firebase.App
var firebaseAuthClient *auth.Client

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

func verifyFirebaseToken(idToken string) (*auth.Token, error) {
    ctx := context.Background()
    token, err := firebaseAuthClient.VerifyIDToken(ctx, idToken)
    if err != nil {
        return nil, err
    }
    return token, nil
}

func userRegisterHandler(w http.ResponseWriter, r *http.Request) {
    type request struct {
        FirebaseToken string `json:"firebase_token"`
        Name          string `json:"name"`
        Email         string `json:"email"`
    }
    var req request
    err := json.NewDecoder(r.Body).Decode(&req)
    if err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    token, err := verifyFirebaseToken(req.FirebaseToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    // Check if user exists
    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil && err != sql.ErrNoRows {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    if err == sql.ErrNoRows {
        // Insert new user
        err = db.QueryRow(
            "INSERT INTO users (phone, name, email) VALUES ($1, $2, $3) RETURNING id",
            phone, req.Name, req.Email).Scan(&userID)
        if err != nil {
            http.Error(w, "Failed to create user", http.StatusInternalServerError)
            return
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "user_id": userID,
        "phone":   phone,
        "name":    req.Name,
        "email":   req.Email,
    })
}

func userProfileHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var user struct {
        ID          int     `json:"id"`
        Phone       string  `json:"phone"`
        Name        *string `json:"name"`
        Email       *string `json:"email"`
        ProfileImage *string `json:"profile_image_url"`
        KYCStatus   string  `json:"kyc_status"`
    }

    err = db.QueryRow("SELECT id, phone, name, email, profile_image_url, kyc_status FROM users WHERE phone=$1", phone).
        Scan(&user.ID, &user.Phone, &user.Name, &user.Email, &user.ProfileImage, &user.KYCStatus)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func uploadKycDocumentHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    type request struct {
        DocumentURL string `json:"document_url"`
    }
    var req request
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil || req.DocumentURL == "" {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    _, err = db.Exec("INSERT INTO kyc_documents (user_id, document_url, status) VALUES ($1, $2, 'pending')", userID, req.DocumentURL)
    if err != nil {
        http.Error(w, "Failed to upload KYC document", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message":"KYC document uploaded successfully"}`))
}

func getKycDocumentsHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    rows, err := db.Query("SELECT id, document_url, status, uploaded_at FROM kyc_documents WHERE user_id=$1", userID)
    if err != nil {
        http.Error(w, "Failed to fetch KYC documents", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    type KycDocument struct {
        ID          int    `json:"id"`
        DocumentURL string `json:"document_url"`
        Status      string `json:"status"`
        UploadedAt  string `json:"uploaded_at"`
    }

    var docs []KycDocument
    for rows.Next() {
        var doc KycDocument
        err := rows.Scan(&doc.ID, &doc.DocumentURL, &doc.Status, &doc.UploadedAt)
        if err != nil {
            http.Error(w, "Error reading KYC documents", http.StatusInternalServerError)
            return
        }
        docs = append(docs, doc)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(docs)
}

func createInvestmentHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    type request struct {
        ProjectID    int     `json:"project_id"`
        Amount       float64 `json:"amount"`
        Reinvest     bool    `json:"reinvest"`
    }
    var req request
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil || req.ProjectID == 0 || req.Amount <= 0 {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get project details for lock_days and profit_percent
    var lockDays int
    var profitPercent float64
    err = db.QueryRow("SELECT lock_days, profit_percent FROM projects WHERE id=$1", req.ProjectID).Scan(&lockDays, &profitPercent)
    if err != nil {
        http.Error(w, "Project not found", http.StatusBadRequest)
        return
    }

    // Calculate lock_end_date
    var lockEndDate string
    err = db.QueryRow("SELECT NOW() + INTERVAL '$1 day'", lockDays).Scan(&lockEndDate)
    if err != nil {
        // fallback to NULL if error
        lockEndDate = ""
    }

    _, err = db.Exec("INSERT INTO investments (user_id, project_id, amount, lock_end_date, profit_percent, reinvest, invested_at) VALUES ($1, $2, $3, NOW() + INTERVAL '$4 day', $5, $6, NOW())",
        userID, req.ProjectID, req.Amount, lockDays, profitPercent, req.Reinvest)
    if err != nil {
        http.Error(w, "Failed to create investment", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message":"Investment created successfully"}`))
}

func listInvestmentsHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    rows, err := db.Query(`
        SELECT i.id, i.amount, i.invested_at, i.lock_end_date, i.profit_percent, i.reinvest, p.name
        FROM investments i
        JOIN projects p ON i.project_id = p.id
        WHERE i.user_id = $1
        ORDER BY i.invested_at DESC
    `, userID)
    if err != nil {
        http.Error(w, "Failed to fetch investments", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    type Investment struct {
        ID           int     `json:"id"`
        Amount       float64 `json:"amount"`
        InvestedAt   string  `json:"invested_at"`
        LockEndDate  string  `json:"lock_end_date"`
        ProfitPercent float64 `json:"profit_percent"`
        Reinvest     bool    `json:"reinvest"`
        ProjectName  string  `json:"project_name"`
    }

    var investments []Investment
    for rows.Next() {
        var inv Investment
        err := rows.Scan(&inv.ID, &inv.Amount, &inv.InvestedAt, &inv.LockEndDate, &inv.ProfitPercent, &inv.Reinvest, &inv.ProjectName)
        if err != nil {
            http.Error(w, "Error reading investments", http.StatusInternalServerError)
            return
        }
        investments = append(investments, inv)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(investments)
}

func createTransactionHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    type request struct {
        ProductID int     `json:"product_id"`
        Type      string  `json:"type"` // buy or sell
        Quantity  float64 `json:"quantity"`
        Unit      string  `json:"unit"` // kg or litre
        Price     float64 `json:"price"`
    }
    var req request
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil || (req.Type != "buy" && req.Type != "sell") || req.Quantity <= 0 || req.Price <= 0 {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Verify product exists
    var productExists bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)", req.ProductID).Scan(&productExists)
    if err != nil || !productExists {
        http.Error(w, "Product not found", http.StatusBadRequest)
        return
    }

    _, err = db.Exec(`
        INSERT INTO transactions 
        (user_id, product_id, type, quantity, unit, price, transaction_date) 
        VALUES ($1, $2, $3, $4, $5, $6, NOW())`,
        userID, req.ProductID, req.Type, req.Quantity, req.Unit, req.Price)
    if err != nil {
        http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message":"Transaction created successfully"}`))
}

func listTransactionsHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    rows, err := db.Query(`
        SELECT t.id, t.type, t.quantity, t.unit, t.price, t.transaction_date, p.name, p.type as product_type
        FROM transactions t
        JOIN products p ON t.product_id = p.id
        WHERE t.user_id = $1
        ORDER BY t.transaction_date DESC
    `, userID)
    if err != nil {
        http.Error(w, "Failed to fetch transactions", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    type Transaction struct {
        ID              int     `json:"id"`
        Type            string  `json:"type"`
        Quantity        float64 `json:"quantity"`
        Unit            string  `json:"unit"`
        Price           float64 `json:"price"`
        TransactionDate string  `json:"transaction_date"`
        ProductName     string  `json:"product_name"`
        ProductType     string  `json:"product_type"`
        TotalAmount     float64 `json:"total_amount"`
    }

    var transactions []Transaction
    for rows.Next() {
        var tr Transaction
        err := rows.Scan(&tr.ID, &tr.Type, &tr.Quantity, &tr.Unit, &tr.Price, &tr.TransactionDate, &tr.ProductName, &tr.ProductType)
        if err != nil {
            http.Error(w, "Error reading transactions", http.StatusInternalServerError)
            return
        }
        tr.TotalAmount = tr.Quantity * tr.Price
        transactions = append(transactions, tr)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(transactions)
}

func createReferralHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    type request struct {
        ReferredPhone string  `json:"referred_phone"`
        Level        int     `json:"level"`
        Commission   float64 `json:"commission"`
    }
    var req request
    err = json.NewDecoder(r.Body).Decode(&req)
    if err != nil || req.Level < 1 || req.Level > 3 || req.Commission < 0 {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Get referred user ID
    var referredUserID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", req.ReferredPhone).Scan(&referredUserID)
    if err != nil {
        http.Error(w, "Referred user not found", http.StatusNotFound)
        return
    }

    // Check if referral already exists
    var exists bool
    err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM referrals WHERE user_id=$1 AND referred_user_id=$2)", 
        userID, referredUserID).Scan(&exists)
    if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }
    if exists {
        http.Error(w, "Referral already exists", http.StatusConflict)
        return
    }

    _, err = db.Exec(`
        INSERT INTO referrals (user_id, referred_user_id, level, commission, created_at)
        VALUES ($1, $2, $3, $4, NOW())`,
        userID, referredUserID, req.Level, req.Commission)
    if err != nil {
        http.Error(w, "Failed to create referral", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte(`{"message":"Referral created successfully"}`))
}

func listReferralsHandler(w http.ResponseWriter, r *http.Request) {
    // Expect Authorization: Bearer <FirebaseToken>
    authHeader := r.Header.Get("Authorization")
    if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
        http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
        return
    }
    idToken := strings.TrimPrefix(authHeader, "Bearer ")

    token, err := verifyFirebaseToken(idToken)
    if err != nil {
        http.Error(w, "Invalid Firebase token", http.StatusUnauthorized)
        return
    }

    phone := token.Claims["phone_number"].(string)

    var userID int
    err = db.QueryRow("SELECT id FROM users WHERE phone=$1", phone).Scan(&userID)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    rows, err := db.Query(`
        WITH RECURSIVE referral_tree AS (
            -- Base case: direct referrals (level 1)
            SELECT r.id, r.user_id, r.referred_user_id, r.level, r.commission, r.created_at,
                   u.phone, u.name, 1 as depth
            FROM referrals r
            JOIN users u ON r.referred_user_id = u.id
            WHERE r.user_id = $1

            UNION ALL

            -- Recursive case: next level referrals
            SELECT r.id, r.user_id, r.referred_user_id, r.level, r.commission, r.created_at,
                   u.phone, u.name, rt.depth + 1
            FROM referrals r
            JOIN users u ON r.referred_user_id = u.id
            JOIN referral_tree rt ON r.user_id = rt.referred_user_id
            WHERE rt.depth < 3  -- Limit to 3 levels
        )
        SELECT * FROM referral_tree
        ORDER BY depth, created_at;
    `, userID)
    if err != nil {
        http.Error(w, "Failed to fetch referrals", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    type Referral struct {
        ID              int     `json:"id"`
        ReferredPhone   string  `json:"referred_phone"`
        ReferredName    string  `json:"referred_name"`
        Level           int     `json:"level"`
        Commission      float64 `json:"commission"`
        CreatedAt       string  `json:"created_at"`
        Depth           int     `json:"depth"`
    }

    var referrals []Referral
    for rows.Next() {
        var r Referral
        var userID, referredUserID int
        err := rows.Scan(&r.ID, &userID, &referredUserID, &r.Level, &r.Commission, &r.CreatedAt,
            &r.ReferredPhone, &r.ReferredName, &r.Depth)
        if err != nil {
            http.Error(w, "Error reading referrals", http.StatusInternalServerError)
            return
        }
        referrals = append(referrals, r)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "referrals": referrals,
        "total_commission": calculateTotalCommission(referrals),
    })
}

func calculateTotalCommission(referrals []Referral) float64 {
    var total float64
    for _, r := range referrals {
        total += r.Commission
    }
    return total
}

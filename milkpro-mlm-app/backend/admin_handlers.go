package main

import (
    "encoding/json"
    "net/http"
)

func adminMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "No authorization header", http.StatusUnauthorized)
            return
        }

        token, err := verifyFirebaseToken(authHeader)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Check if user is admin
        var isAdmin bool
        err = db.QueryRow("SELECT is_admin FROM users WHERE id = $1", token.UID).Scan(&isAdmin)
        if err != nil || !isAdmin {
            http.Error(w, "Unauthorized", http.StatusForbidden)
            return
        }

        next.ServeHTTP(w, r)
    }
}

func getDashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
    var stats struct {
        TotalUsers        int     `json:"total_users"`
        PendingKYC       int     `json:"pending_kyc"`
        TotalInvestments float64 `json:"total_investments"`
        TotalProducts    int     `json:"total_products"`
    }

    err := db.QueryRow(`
        SELECT 
            (SELECT COUNT(*) FROM users),
            (SELECT COUNT(*) FROM users WHERE kyc_status = 'pending'),
            COALESCE((SELECT SUM(amount) FROM investments), 0),
            (SELECT COUNT(*) FROM products)
    `).Scan(&stats.TotalUsers, &stats.PendingKYC, &stats.TotalInvestments, &stats.TotalProducts)

    if err != nil {
        http.Error(w, "Failed to fetch dashboard stats", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    rows, err := db.Query(`
        SELECT 
            u.id, u.phone, u.name, u.email, u.kyc_status,
            COALESCE((SELECT SUM(amount) FROM investments WHERE user_id = u.id), 0) as total_invested,
            COALESCE((SELECT COUNT(*) FROM referrals WHERE user_id = u.id), 0) as total_referrals
        FROM users u
        ORDER BY u.created_at DESC
    `)
    if err != nil {
        http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var users []struct {
        ID             int     `json:"id"`
        Phone         string  `json:"phone"`
        Name          *string `json:"name"`
        Email         *string `json:"email"`
        KYCStatus     string  `json:"kyc_status"`
        TotalInvested float64 `json:"total_invested"`
        TotalReferrals int    `json:"total_referrals"`
    }

    for rows.Next() {
        var user struct {
            ID             int     `json:"id"`
            Phone         string  `json:"phone"`
            Name          *string `json:"name"`
            Email         *string `json:"email"`
            KYCStatus     string  `json:"kyc_status"`
            TotalInvested float64 `json:"total_invested"`
            TotalReferrals int    `json:"total_referrals"`
        }
        err := rows.Scan(&user.ID, &user.Phone, &user.Name, &user.Email, &user.KYCStatus,
            &user.TotalInvested, &user.TotalReferrals)
        if err != nil {
            http.Error(w, "Error reading users", http.StatusInternalServerError)
            return
        }
        users = append(users, user)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(users)
}

func updateKycStatusHandler(w http.ResponseWriter, r *http.Request) {
    var req struct {
        UserID int    `json:"user_id"`
        Status string `json:"status"` // approved or rejected
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    if req.Status != "approved" && req.Status != "rejected" {
        http.Error(w, "Invalid status", http.StatusBadRequest)
        return
    }

    result, err := db.Exec("UPDATE users SET kyc_status = $1 WHERE id = $2", req.Status, req.UserID)
    if err != nil {
        http.Error(w, "Failed to update KYC status", http.StatusInternalServerError)
        return
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil || rowsAffected == 0 {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"message": "KYC status updated successfully"})
}

func manageProductHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        rows, err := db.Query("SELECT id, name, type, price FROM products ORDER BY type, name")
        if err != nil {
            http.Error(w, "Failed to fetch products", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var products []struct {
            ID    int     `json:"id"`
            Name  string  `json:"name"`
            Type  string  `json:"type"`
            Price float64 `json:"price"`
        }

        for rows.Next() {
            var p struct {
                ID    int     `json:"id"`
                Name  string  `json:"name"`
                Type  string  `json:"type"`
                Price float64 `json:"price"`
            }
            if err := rows.Scan(&p.ID, &p.Name, &p.Type, &p.Price); err != nil {
                http.Error(w, "Error reading products", http.StatusInternalServerError)
                return
            }
            products = append(products, p)
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(products)

    case "POST":
        var product struct {
            Name  string  `json:"name"`
            Type  string  `json:"type"`
            Price float64 `json:"price"`
        }

        if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        var productID int
        err := db.QueryRow(
            "INSERT INTO products (name, type, price) VALUES ($1, $2, $3) RETURNING id",
            product.Name, product.Type, product.Price,
        ).Scan(&productID)

        if err != nil {
            http.Error(w, "Failed to create product", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id": productID,
            "message": "Product created successfully",
        })
    }
}

func manageProjectHandler(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
    case "GET":
        rows, err := db.Query(`
            SELECT 
                id, name, description, lock_days, profit_percent, 
                min_investment, max_investment, status
            FROM projects 
            ORDER BY created_at DESC
        `)
        if err != nil {
            http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        var projects []struct {
            ID            int     `json:"id"`
            Name          string  `json:"name"`
            Description   string  `json:"description"`
            LockDays      int     `json:"lock_days"`
            ProfitPercent float64 `json:"profit_percent"`
            MinInvestment float64 `json:"min_investment"`
            MaxInvestment float64 `json:"max_investment"`
            Status        string  `json:"status"`
        }

        for rows.Next() {
            var p struct {
                ID            int     `json:"id"`
                Name          string  `json:"name"`
                Description   string  `json:"description"`
                LockDays      int     `json:"lock_days"`
                ProfitPercent float64 `json:"profit_percent"`
                MinInvestment float64 `json:"min_investment"`
                MaxInvestment float64 `json:"max_investment"`
                Status        string  `json:"status"`
            }
            if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.LockDays,
                &p.ProfitPercent, &p.MinInvestment, &p.MaxInvestment, &p.Status); err != nil {
                http.Error(w, "Error reading projects", http.StatusInternalServerError)
                return
            }
            projects = append(projects, p)
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(projects)

    case "POST":
        var project struct {
            Name          string  `json:"name"`
            Description   string  `json:"description"`
            LockDays      int     `json:"lock_days"`
            ProfitPercent float64 `json:"profit_percent"`
            MinInvestment float64 `json:"min_investment"`
            MaxInvestment float64 `json:"max_investment"`
        }

        if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
            http.Error(w, "Invalid request body", http.StatusBadRequest)
            return
        }

        var projectID int
        err := db.QueryRow(`
            INSERT INTO projects 
            (name, description, lock_days, profit_percent, min_investment, max_investment, status, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, 'active', NOW())
            RETURNING id
        `, project.Name, project.Description, project.LockDays,
            project.ProfitPercent, project.MinInvestment, project.MaxInvestment,
        ).Scan(&projectID)

        if err != nil {
            http.Error(w, "Failed to create project", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id": projectID,
            "message": "Project created successfully",
        })
    }
}

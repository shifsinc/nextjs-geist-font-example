package main

import (
    "encoding/json"
    "html/template"
    "log"
    "net/http"
    "path/filepath"
    "github.com/gorilla/sessions"
    "github.com/gorilla/websocket"
)

var (
    store = sessions.NewCookieStore([]byte("secret-key-replace-in-production"))
    templates *template.Template
    upgrader = websocket.Upgrader{
        ReadBufferSize:  1024,
        WriteBufferSize: 1024,
        CheckOrigin: func(r *http.Request) bool {
            return true // In production, implement proper origin checks
        },
    }
)

type PageData struct {
    Title        string
    Active       string
    Stats        *DashboardStats
    ChartData    *ChartDataPoint
    Error        string
    User         *User
    Tickets      []Ticket
    ChatSessions []ChatSession
    Session      *ChatSession
    Messages     []ChatMessage
}

type User struct {
    ID       int
    Username string
    Role     string
}

type DashboardStats struct {
    TotalUsers        int     `json:"total_users"`
    TotalInvestments float64 `json:"total_investments"`
    TotalTransactions float64 `json:"total_transactions"`
    PendingKYC       int     `json:"pending_kyc"`
}

type ChartDataPoint struct {
    Labels []string  `json:"labels"`
    Values []float64 `json:"values"`
}

type Ticket struct {
    ID        int    `json:"id"`
    Subject   string `json:"subject"`
    UserName  string `json:"user_name"`
    Status    string `json:"status"`
    Priority  string `json:"priority"`
    CreatedAt string `json:"created_at"`
}

type ChatSession struct {
    ID        int    `json:"id"`
    UserName  string `json:"user_name"`
    UserEmail string `json:"user_email"`
    Status    string `json:"status"`
    CreatedAt string `json:"created_at"`
}

type ChatMessage struct {
    SessionID  int    `json:"session_id"`
    SenderType string `json:"sender_type"`
    Message    string `json:"message"`
    CreatedAt  string `json:"created_at"`
}

func main() {
    templates = loadTemplates()

    // Authentication middleware
    authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            session, _ := store.Get(r, "admin-session")
            if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
                http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
                return
            }
            next.ServeHTTP(w, r)
        }
    }

    // Login routes
    http.HandleFunc("/admin/login", handleLogin)
    http.HandleFunc("/admin/logout", handleLogout)

    // Protected routes
    http.HandleFunc("/admin/dashboard", authMiddleware(handleDashboard))
    http.HandleFunc("/admin/support", authMiddleware(handleSupport))
    http.HandleFunc("/admin/chat/", authMiddleware(handleChat))
    http.HandleFunc("/ws/chat/", authMiddleware(handleWebSocket))

    // Serve static files
    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    log.Println("Starting admin panel on :8000")
    log.Fatal(http.ListenAndServe(":8000", nil))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "admin-session")
    
    if auth, ok := session.Values["authenticated"].(bool); ok && auth {
        http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
        return
    }

    if r.Method == "POST" {
        username := r.FormValue("username")
        password := r.FormValue("password")

        // TODO: Replace with actual database authentication
        if username == "admin" && password == "admin" {
            session.Values["authenticated"] = true
            session.Values["user"] = &User{
                ID:       1,
                Username: username,
                Role:     "admin",
            }
            session.Save(r, w)
            http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
            return
        }

        templates.ExecuteTemplate(w, "login.html", PageData{
            Error: "Invalid username or password",
        })
        return
    }

    templates.ExecuteTemplate(w, "login.html", nil)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "admin-session")
    session.Values["authenticated"] = false
    session.Values["user"] = nil
    session.Save(r, w)
    http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "admin-session")
    user := session.Values["user"].(*User)

    stats := &DashboardStats{
        TotalUsers:        100,
        TotalInvestments: 50000.00,
        TotalTransactions: 75000.00,
        PendingKYC:       5,
    }

    chartData := &ChartDataPoint{
        Labels: []string{"Jan", "Feb", "Mar", "Apr", "May"},
        Values: []float64{10000, 15000, 20000, 25000, 30000},
    }

    data := PageData{
        Title:     "Dashboard",
        Active:    "dashboard",
        Stats:     stats,
        ChartData: chartData,
        User:      user,
    }

    templates.ExecuteTemplate(w, "layout.html", data)
}

func handleSupport(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:  "Support",
        Active: "support",
        Tickets: []Ticket{
            {
                ID:        1,
                Subject:   "Payment Issue",
                UserName:  "John Doe",
                Status:    "open",
                Priority:  "high",
                CreatedAt: "2024-05-26 15:30:00",
            },
        },
        ChatSessions: []ChatSession{
            {
                ID:        1,
                UserName:  "Jane Smith",
                UserEmail: "jane@example.com",
                Status:    "active",
                CreatedAt: "2024-05-26 15:45:00",
            },
        },
    }

    templates.ExecuteTemplate(w, "support.html", data)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
    data := PageData{
        Title:  "Chat Session",
        Active: "support",
        Session: &ChatSession{
            ID:        1,
            UserName:  "Jane Smith",
            UserEmail: "jane@example.com",
            Status:    "active",
            CreatedAt: "2024-05-26 15:45:00",
        },
        Messages: []ChatMessage{
            {
                SessionID:  1,
                SenderType: "user",
                Message:    "Hello, I need help with my order",
                CreatedAt:  "2024-05-26 15:45:00",
            },
        },
    }

    templates.ExecuteTemplate(w, "chat.html", data)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade error: %v", err)
        return
    }
    defer conn.Close()

    for {
        messageType, p, err := conn.ReadMessage()
        if err != nil {
            log.Printf("WebSocket read error: %v", err)
            return
        }

        // Echo the message back
        if err := conn.WriteMessage(messageType, p); err != nil {
            log.Printf("WebSocket write error: %v", err)
            return
        }
    }
}

func loadTemplates() *template.Template {
    templatesDir := "templates"
    pattern := filepath.Join(templatesDir, "*.html")
    
    log.Printf("Loading templates from: %s", pattern)
    
    funcMap := template.FuncMap{
        "safeJS": func(v interface{}) template.JS {
            b, err := json.Marshal(v)
            if err != nil {
                return template.JS("{}")
            }
            return template.JS(string(b))
        },
    }
    
    tmpl, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
    if err != nil {
        log.Fatalf("Error loading templates: %v", err)
    }
    
    for _, t := range tmpl.Templates() {
        log.Printf("Loaded template: %s", t.Name())
    }
    
    return tmpl
}

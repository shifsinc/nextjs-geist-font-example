package main

import (
    "encoding/json"
    "html/template"
    "log"
    "net/http"
    "path/filepath"
    "time"
)

type User struct {
    ID        int
    Name      string
    Email     string
    Phone     string
    CreatedAt time.Time
}

type Product struct {
    ID          int
    Name        string
    Description string
    Price       float64
}

type PageData struct {
    Title    string
    Active   string
    Stats    *DashboardStats
    ChartData *ChartData
    Users    []User
    Products []Product
}

type DashboardStats struct {
    TotalUsers        int     `json:"total_users"`
    TotalInvestments float64 `json:"total_investments"`
    TotalTransactions float64 `json:"total_transactions"`
    PendingKYC       int     `json:"pending_kyc"`
}

type ChartData struct {
    InvestmentsData    ChartDataPoint
    TransactionsData   ChartDataPoint
}

func main() {
    // Load templates
    templates := loadTemplates()

    // Root route redirect to dashboard
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" {
            http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
            return
        }
        http.NotFound(w, r)
    })

    http.HandleFunc("/admin/dashboard", func(w http.ResponseWriter, r *http.Request) {
        stats := &DashboardStats{
            TotalUsers:        100,  // TODO: Get from database
            TotalInvestments: 50000.00,
            TotalTransactions: 75000.00,
            PendingKYC:       5,
        }

        // Sample chart data
        chartData := &ChartData{
            InvestmentsData: ChartDataPoint{
                Labels: []string{"Jan", "Feb", "Mar", "Apr", "May"},
                Values: []float64{10000, 15000, 20000, 25000, 30000},
            },
            TransactionsData: ChartDataPoint{
                Labels: []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
                Values: []float64{1000, 1500, 2000, 1800, 2200},
            },
        }

        data := PageData{
            Title:     "Dashboard",
            Active:    "dashboard",
            Stats:     stats,
            ChartData: chartData,
            Users:     []User{},
            Products:  []Product{},
        }

        err := templates.ExecuteTemplate(w, "layout.html", data)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    })

    // Serve static files
    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    log.Println("Starting admin panel on :8000")
    log.Fatal(http.ListenAndServe(":8000", nil))
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
    
    // List loaded templates for debugging
    for _, t := range tmpl.Templates() {
        log.Printf("Loaded template: %s", t.Name())
    }
    
    return tmpl
}

type ChartDataPoint struct {
    Labels []string    `json:"labels"`
    Values []float64   `json:"values"`
}

func (c ChartData) ToJSON() template.JS {
    b, err := json.Marshal(c)
    if err != nil {
        return template.JS("{}")
    }
    return template.JS(string(b))
}

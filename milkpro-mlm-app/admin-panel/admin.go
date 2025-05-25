package main

import (
    "encoding/json"
    "html/template"
    "log"
    "net/http"
    "path/filepath"
)

type PageData struct {
    Title    string
    Active   string
    Stats    *DashboardStats
    ChartData *ChartData
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
    templates := loadTemplates()

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
        }

        err := templates.ExecuteTemplate(w, "dashboard.html", data)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    })

    // Serve static files
    fs := http.FileServer(http.Dir("static"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

    log.Println("Starting admin panel on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadTemplates() *template.Template {
    templatesDir := "templates"
    pattern := filepath.Join(templatesDir, "*.html")
    
    funcMap := template.FuncMap{
        "safeJS": func(v interface{}) template.JS {
            b, err := json.Marshal(v)
            if err != nil {
                return template.JS("{}")
            }
            return template.JS(string(b))
        },
    }
    
    return template.Must(template.New("").Funcs(funcMap).ParseGlob(pattern))
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

package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type patient struct {
	FirstNameTH  string `json:"first_name_th"`
	MiddleNameTH string `json:"middle_name_th"`
	LastNameTH   string `json:"last_name_th"`
	FirstNameEN  string `json:"first_name_en"`
	MiddleNameEN string `json:"middle_name_en"`
	LastNameEN   string `json:"last_name_en"`
	DateOfBirth  string `json:"date_of_birth"`
	PatientHN    string `json:"patient_hn"`
	NationalID   string `json:"national_id"`
	PassportID   string `json:"passport_id"`
	PhoneNumber  string `json:"phone_number"`
	Email        string `json:"email"`
	Gender       string `json:"gender"`
}

// Mock data.
var patients = []patient{
	{
		PatientHN: "HN9001", FirstNameTH: "วิชัย", LastNameTH: "มั่นคง",
		FirstNameEN: "Wichai", LastNameEN: "Mankong", DateOfBirth: "1988-03-14",
		NationalID: "1100500077777", PhoneNumber: "0871234567", Email: "wichai@example.com", Gender: "M",
	},
	{
		PatientHN: "HN9002", FirstNameTH: "เอ็มม่า", MiddleNameTH: "โรส", LastNameTH: "จอห์นสัน",
		FirstNameEN: "Emma", MiddleNameEN: "Rose", LastNameEN: "Johnson", DateOfBirth: "1993-12-01",
		PassportID: "CD9876543", PhoneNumber: "0869876543", Email: "emma.j@example.com", Gender: "F",
	},
}

func main() {
	byID := make(map[string]patient)
	for _, p := range patients {
		if p.NationalID != "" {
			byID[p.NationalID] = p
		}
		if p.PassportID != "" {
			byID[p.PassportID] = p
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /patient/search/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		slog.Info("search", "id", id)

		switch id {
		case failingID:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		case slowID:
			time.Sleep(10 * time.Second)
		}

		p, ok := byID[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "patient not found"})
			return
		}
		writeJSON(w, http.StatusOK, p)
	})

	addr := ":" + getEnv("MOCKHIS_PORT", "8081")
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	slog.Info("mock HIS listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("mock HIS failed", "error", err)
		os.Exit(1)
	}
}

// Demo error codes.
const (
	failingID = "0000000000500" // responds 500
	slowID    = "0000000000504" // responds after 10s, longer than the client timeout
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

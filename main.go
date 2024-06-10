package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.Use(enableCors)
	r.HandleFunc("/upload", uploadHandler).Methods("POST", "OPTIONS")

	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", r)
}

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max upload size
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	// Create the uploads folder if it doesn't
	// already exist
	err = os.MkdirAll("./uploads", os.ModePerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new file in the uploads directory
	dst, err := os.Create(fmt.Sprintf("./uploads/%d%s", time.Now().UnixNano(), fileHeader.Filename))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	// READ FILE'S CONTENT
	file.Seek(0, io.SeekStart) // Reset the file pointer to the beginning
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read uploaded file", http.StatusInternalServerError)
		return
	}
	fileContent := string(fileBytes)

	// DETECT PASSWORD USING REGEX
	passwordRegex := regexp.MustCompile(`\b\S{8,20}\b`)
	passwords := passwordRegex.FindAllString(fileContent, -1)
	
	if len(passwords) > 0 {

		// INSERT FILE INTO ELASTIC SERVICE
		fmt.Fprintf(w, "Detected passwords:\n%s", passwords)
	} else {
		fmt.Fprintf(w, "No passwords detected in the uploaded file")
	}

	// Copy the uploaded file to the file on the server
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save uploaded file", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %s", fileHeader.Filename)
}

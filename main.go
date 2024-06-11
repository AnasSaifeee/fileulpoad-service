package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
	

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
    r := mux.NewRouter()

    // Define your routes
    r.HandleFunc("/upload", uploadHandler).Methods("POST", "OPTIONS")

    // Wrap the router with CORS middleware
    loggedRouter := handlers.LoggingHandler(os.Stdout, r)
    corsHandler := handlers.CORS(
        handlers.AllowedHeaders([]string{"*"}),
        handlers.AllowedMethods([]string{"*"}),
        handlers.AllowedOrigins([]string{"*"}),
    )(loggedRouter)

    // Create the server with timeouts
    server := &http.Server{
        Addr:         ":8080",
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
        IdleTimeout:  60 * time.Second,
        Handler:      corsHandler, // Use the CORS handler
    }

    fmt.Println("Server is running on port 8080...")
    if err := server.ListenAndServe(); err != nil {
        fmt.Printf("Failed to start server: %v\n", err)
    }
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
		fileInfo := ClassificationLogModel{
			Id:                 fmt.Sprintf("disc%d", time.Now().UnixNano()),
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			TenantId:           1001,
			JobID:              196,
			Asset:              fileHeader.Filename,
			ParentAsset:        "Inline-SampleApp",
			SourceType:         "SampleApp",
			RootAsset:          "SampleApp/Folder",
			ClassificationType: "regex",
			InfoType:           "file-info",
			FileIdentifiers:    passwords,
			Identifiers:        nil,
			FileSizeInBytes:    fileHeader.Size,
			AgentID:            100,
			BlockNum:           -1,
			LastAccessedAt:     "",
			LastModifiedAt:     time.Now().UTC().Format(time.RFC3339),
			Labels: []map[string]interface{}{
				{
					"identifiers": "Password",
					"name":        "Password-policy-test",
				},
			},
			RunId: 1,
		}
		fmt.Println("file info is",fileInfo)
		jsonData, err := json.Marshal(fileInfo)
		if err != nil {
			http.Error(w, "Failed to create JSON response", http.StatusInternalServerError)
			return
		}
		
		endpointURL := "http://localhost:8098/elastic/classification"
		resp, err := http.Post(endpointURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			http.Error(w, "Failed to send JSON to endpoint", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			http.Error(w, fmt.Sprintf("Failed to send JSON to endpoint: %s", body), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
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

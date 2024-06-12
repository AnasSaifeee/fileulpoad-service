package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func main() {
	r := mux.NewRouter()
	r.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Authorization", "*")
		w.Header().Set("Content-Type", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.WriteHeader(http.StatusNoContent)
		return
	})

	// Define your routes
	r.HandleFunc("/upload", uploadHandler).Methods("POST", "OPTIONS")
	s := r.PathPrefix("/").Subrouter()
	s.Use(mux.CORSMethodMiddleware(s))
	// Wrap the router with CORS middleware
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	handler := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "DELETE", "PATCH", "PUT"},
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		Debug:            true,
	}).Handler(loggedRouter)

	// Create the server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
		Handler:      handler, // Use the CORS handler
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
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err = os.Mkdir(uploadDir, 0755)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create a new file in the uploads directory
// Construct the file path
originalFilename := fileHeader.Filename
filePath := filepath.Join(uploadDir, originalFilename)

// Check if a file with the same name already exists and add suffix if necessary
counter := 1
for {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		break
	}
	ext := filepath.Ext(originalFilename)
	name := strings.TrimSuffix(originalFilename, ext)
	filePath = filepath.Join(uploadDir, fmt.Sprintf("%s(%d)%s", name, counter, ext))
	counter++
}

// Create the new file
dst, err := os.Create(filePath)
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
defer dst.Close()


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
		
		// identifiersList:=[]map[string]interface{}{}

		// for _,val:= range passwords{

		// 	newBlock:=map[string]interface{}{
		// 		"Password":val,
		// 	}
		// 	identifiersList=append(identifiersList, newBlock)
		// }
		// // INSERT FILE INTO ELASTIC SERVICE
		// blockInfo := ClassificationLogModel{
		// 	Id:                 fmt.Sprintf("disc%d", time.Now().UnixNano()),
		// 	Timestamp:          time.Now().UTC().Format(time.RFC3339),
		// 	TenantId:           1001,
		// 	JobID:              196,
		// 	Asset:              fileHeader.Filename,
		// 	ParentAsset:        "Inline-SampleApp",
		// 	SourceType:         "uploadservice",
		// 	RootAsset:          "SampleApp/Folder",
		// 	ClassificationType: "regex",
		// 	InfoType:           "block-info",
		// 	// FileIdentifiers:    []string{"Password"},
		// 	Identifiers:        identifiersList,
		// 	FileSizeInBytes:    fileHeader.Size,
		// 	AgentID:            100,
		// 	BlockNum:           1,
		// 	LastAccessedAt:     time.Now().UTC().Format(time.RFC3339),
		// 	LastModifiedAt:     time.Now().UTC().Format(time.RFC3339),
		// 	RunId: 1,
		// }

		fmt.Fprintf(w, "Detected passwords:\n%s", passwords)
		fileInfo := ClassificationLogModel{
			Id:                 fmt.Sprintf("disc%d", time.Now().UnixNano()),
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			TenantId:           1001,
			JobID:              196,
			Asset:              fileHeader.Filename,
			ParentAsset:        "Inline-SampleApp",
			SourceType:         "uploadservice",
			RootAsset:          "SampleApp/Folder",
			ClassificationType: "regex",
			InfoType:           "file-info",
			FileIdentifiers:    []string{"Password"},
			Identifiers:        nil,
			FileSizeInBytes:    fileHeader.Size,
			AgentID:            100,
			BlockNum:           -1,
			LastAccessedAt:     time.Now().UTC().Format(time.RFC3339),
			LastModifiedAt:     time.Now().UTC().Format(time.RFC3339),
			Labels: []map[string]interface{}{
				{
					"identifiers": "Password",
					"name":        "Password-policy-test",
				},
			},
			RunId: 1,
		}
		
		
		fileInfoArr := []ClassificationLogModel{fileInfo}
	
		// SYNC FUNCTION LOGIC
		es, err := NewOpenSearchService()
		if err != nil {
			http.Error(w, "Error while creating new elastic service", http.StatusInternalServerError)
			return
		}

		err = es.CreateBulkClassificationRecords(fileInfoArr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"message": "Sync Successful with elastic Server",
			"data":    nil,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
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

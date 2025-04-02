package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/DigitalArsenal/spacedatastandards.org/lib/go/EPM"
	flatbuffers "github.com/google/flatbuffers/go"
)

const (
	EPMFID       = "$EPM"          // FlatBuffer file identifier (4 bytes)
	lastFilePath = "last_stack.fb" // File to store the last transmitted stack
)

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/stream", streamHandler)
	http.HandleFunc("/submit", submitHandler)
	http.HandleFunc("/last", lastHandler)

	fmt.Println("Server listening on :8080")
	http.ListenAndServe(":8080", nil)
}

// streamHandler generates a stream of EPM flatbuffers.
// It accepts an optional "count" query parameter (default 1000).
// Each message is built with random values for DN, LEGAL_NAME, EMAIL, and TELEPHONE.
// The stream is written to the HTTP response and saved (overwritten) to disk.
func streamHandler(w http.ResponseWriter, r *http.Request) {
	count := 1000
	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil {
			count = n
		}
	}

	// Create or overwrite the file for the last transmitted stack.
	f, err := os.Create(lastFilePath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")

	for i := 0; i < count; i++ {
		randomNum := rand.Intn(10000)
		dn := fmt.Sprintf("DN-%d", randomNum)
		legalName := fmt.Sprintf("LegalName-%d", randomNum)
		email := fmt.Sprintf("user%d@example.com", randomNum)
		telephone := fmt.Sprintf("+1-555-%04d", randomNum)

		data := CreateEPM(dn, legalName, email, telephone)

		// Write the flatbuffer message to both HTTP response and disk.
		if _, err := w.Write(data); err != nil {
			fmt.Println("Error writing to response:", err)
			return
		}
		if _, err := f.Write(data); err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}
}

// submitHandler processes a POST request containing a stream of EPM flatbuffers.
// It overwrites the stored file and prints selected fields to the console.
func submitHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Overwrite the last transmitted file.
	if err := ioutil.WriteFile(lastFilePath, data, 0644); err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}

	processStream(data)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Data processed"))
}

// lastHandler reads the last transmitted stack of flatbuffers from disk
// and prints selected fields to the console.
func lastHandler(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(lastFilePath)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	processStream(data)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Last file processed"))
}

// processStream reads a stream of flatbuffer messages from data.
// Each message is expected to be size-prefixed with the file identifier as per the FlatBuffer spec.
// For each message, it prints VERSION, DN, LEGAL NAME, EMAIL, and TELEPHONE.
func processStream(data []byte) {
	offset := 0
	for offset < len(data) {
		if offset+4 > len(data) {
			fmt.Println("Incomplete size prefix")
			break
		}
		// Read the size prefix (uint32, little endian)
		msgSize := binary.LittleEndian.Uint32(data[offset : offset+4])
		totalMsgSize := int(4 + msgSize)
		if offset+totalMsgSize > len(data) {
			fmt.Println("Incomplete message")
			break
		}
		msg := data[offset : offset+totalMsgSize]
		offset += totalMsgSize

		epm := EPM.GetSizePrefixedRootAsEPM(msg, 0)
		dn := string(epm.DN())
		legalName := string(epm.LEGAL_NAME())
		email := string(epm.EMAIL())
		telephone := string(epm.TELEPHONE())

		fmt.Printf("VERSION: %s\nDN: %s\nLEGAL NAME: %s\nEMAIL: %s\nTELEPHONE: %s\n\n",
			EPMFID, dn, legalName, email, telephone)
	}
}

// CreateEPM builds an EPM flatbuffer message with the provided field values.
// It sets the DN, LEGAL_NAME, EMAIL, and TELEPHONE fields, and finishes the buffer
// using the size-prefixed file identifier method.
func CreateEPM(dn, legalName, email, telephone string) []byte {
	builder := flatbuffers.NewBuilder(0)

	dnOffset := builder.CreateString(dn)
	legalNameOffset := builder.CreateString(legalName)
	emailOffset := builder.CreateString(email)
	telephoneOffset := builder.CreateString(telephone)

	EPM.EPMStart(builder)
	EPM.EPMAddDN(builder, dnOffset)
	EPM.EPMAddLEGAL_NAME(builder, legalNameOffset)
	EPM.EPMAddEMAIL(builder, emailOffset)
	EPM.EPMAddTELEPHONE(builder, telephoneOffset)
	epm := EPM.EPMEnd(builder)

	builder.FinishSizePrefixedWithFileIdentifier(epm, []byte(EPMFID))
	return builder.FinishedBytes()
}

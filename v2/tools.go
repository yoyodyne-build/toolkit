package toolkit

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// randomStringSource is a string of characters used to generate random strings
const randomStringSource = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_+"

// Tools is a struct that contains useful utilities for applications
type Tools struct {
	MaxFileSize        int64
	AllowedFileTypes   []string
	MaxJSONSize        int64
	AllowUnknownFields bool
}

// CheckFileType checks if a file type is allowed
func (t *Tools) CheckFileType(fileType string) bool {
	if len(t.AllowedFileTypes) == 0 {
		t.AllowedFileTypes = []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "application/pdf"}
	}

	for _, t := range t.AllowedFileTypes {
		if strings.EqualFold(fileType, t) {
			return true
		}
	}

	return false
}

// CreateDirIfNotExist creates a directory, and all necessary parent directories, if they do not exist
func (t *Tools) CreateDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

// DownloadStaticFile sends file to the client and attempts to force the browser to download the file,
// saving it as the value provided in the displayName parameter
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, pathName, displayName string) {
	if _, err := os.Stat(pathName); os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(w, r, pathName)
}

// GetNewFileName generates a new file name
func (t *Tools) GetNewFileName(fileHeader *multipart.FileHeader, renameFile bool) string {
	if renameFile {
		return fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(fileHeader.Filename))
	}

	return fileHeader.Filename
}

// HandleFile processes a single file and returns an UploadedFile and an error
func (t *Tools) HandleFile(fileHeader *multipart.FileHeader, uploadDir string, renameFile bool) (*UploadedFile, error) {
	var uploadedFile UploadedFile
	infile, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer infile.Close()

	buff := make([]byte, 512)
	_, err = infile.Read(buff)
	if err != nil {
		return nil, err
	}

	// check to see if file type is permitted
	fileType := http.DetectContentType(buff)
	if !t.CheckFileType(fileType) {
		return nil, errors.New("file type not permitted")
	}

	// be kind, rewind
	_, err = infile.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	uploadedFile.OriginalFileName = fileHeader.Filename
	uploadedFile.NewFileName = t.GetNewFileName(fileHeader, renameFile)

	var outfile *os.File
	defer outfile.Close()

	if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
		return nil, err
	}
	fileSize, err := io.Copy(outfile, infile)
	if err != nil {
		return nil, err
	}
	uploadedFile.FileSize = fileSize

	return &uploadedFile, nil
}

// RandomString generates a random string of length n
func (t *Tools) RandomString(length int) string {
	s, r := make([]rune, length), []rune(randomStringSource)

	for i := range s {
		p, _ := rand.Prime(rand.Reader, len(r))
		x, y := p.Uint64(), uint64(len(r))
		s[i] = r[x%y]
	}

	return string(s)
}

// UploadedFile is used to save information about an uploaded file
type UploadedFile struct {
	NewFileName      string
	OriginalFileName string
	FileSize         int64
}

// Slugify converts string s into an URL safe slug
func (t *Tools) Slugify(s string) (string, error) {
	if strings.Trim(s, " ") == "" {
		return "", errors.New("empty string not permitted")
	}

	var re = regexp.MustCompile(`[^a-z\d]+`)

	slug := strings.Trim(re.ReplaceAllString(strings.ToLower(s), "-"), "-")
	if len(slug) == 0 {
		return "", errors.New("slugified string is empty")
	}

	return slug, nil
}

// UploadFile is a convenience method that call UploadFiles, but expects only one file.
// It returns an UploadedFile and potentially an error.
// If the optional last parameter is set to false, the original file name will be used.
func (t *Tools) UploadFile(r *http.Request, uploadDir string, rename ...bool) (*UploadedFile, error) {
	renameFile := true

	if len(rename) > 0 {
		renameFile = rename[0]
	}

	files, err := t.UploadFiles(r, uploadDir, renameFile)
	if err != nil {
		return nil, err
	}

	return files[0], nil
}

// UploadFiles uploads one or more files to a specified directory, and gives the file a random name.
// It returns a slice of UploadedFile and potentially an error.
// If the optional last parameter is set to false, the original file name will be used.
func (t *Tools) UploadFiles(r *http.Request, uploadDir string, rename ...bool) ([]*UploadedFile, error) {
	renameFile := true

	if len(rename) > 0 {
		renameFile = rename[0]
	}

	var uploadedFiles []*UploadedFile

	if t.MaxFileSize == 0 {
		t.MaxFileSize = 1024 * 1024 * 1024
	}

	err := t.CreateDirIfNotExist(uploadDir)
	if err != nil {
		return nil, err
	}

	err = r.ParseMultipartForm(t.MaxFileSize)
	if err != nil {
		return nil, errors.New("the uploaded file is too big")
	}

	for _, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			uploadedFile, err := t.HandleFile(fileHeader, uploadDir, renameFile)
			if err != nil {
				return uploadedFiles, err
			}
			uploadedFiles = append(uploadedFiles, uploadedFile)
		}
	}

	return uploadedFiles, nil
}

// JSONResponse defines the contract for a JSON response
type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// decodeJSON takes a http.Request and data interface{} and returns an error.
func (t *Tools) decodeJSON(r *http.Request, data interface{}) error {
	decoder := json.NewDecoder(r.Body)
	if !t.AllowUnknownFields {
		decoder.DisallowUnknownFields()
	}

	err := decoder.Decode(data)
	if err != nil {
		return err
	}

	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON payload")
	}

	return nil
}

// handleError takes an error and returns a formatted error message.
func (t *Tools) handleError(err error, maxBytes int64) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	unknownFieldError := "json: unknown field"

	switch {
	case errors.As(err, &syntaxError):
		return fmt.Errorf("body contains badly formed JSON at character %d", syntaxError.Offset)

	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field == "" {
			return fmt.Errorf("body contains badly formed JSON type for field %q", unmarshalTypeError.Field)
		}
		return fmt.Errorf("body contains badly formed JSON at character %d", unmarshalTypeError.Offset)

	case errors.As(err, &invalidUnmarshalError):
		return fmt.Errorf("error unmarshalling JSON: %s", err.Error())

	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("body contains badly formed JSON (unexpected EOF marker)")

	case errors.Is(err, io.EOF):
		return errors.New("body must not be empty")

	case strings.HasPrefix(err.Error(), unknownFieldError):
		fieldName := strings.TrimPrefix(err.Error(), unknownFieldError)
		return fmt.Errorf("body contains unknown field %s", fieldName)

	case err.Error() == "http: request body too large":
		return fmt.Errorf("body must not be larger than %d bytes", maxBytes)

	default:
		return err
	}
}

// ReadJSON attempts to convert the body of a request from JSON to a struct
func (t *Tools) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	// prevent malicious content size
	maxBytes := int64(1024 * 10243)
	if t.MaxJSONSize != 0 {
		maxBytes = t.MaxJSONSize
	}

	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)

	err := t.decodeJSON(r, data)
	if err != nil {
		return t.handleError(err, maxBytes)
	}

	return nil
}

// WriteJSON takes a response status and arbitrary data and writes JSON to the client
func (t *Tools) WriteJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, err = w.Write(out)
	return err
}

// ErrorJSON takes an error and optionally a status code and sends a formatted JSON error to the client
func (t *Tools) ErrorJSON(w http.ResponseWriter, err error, status ...int) error {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()

	return t.WriteJSON(w, statusCode, payload)
}

// PostJSONToRemote posts arbitrary JSON data to the specified uri and returns the response, status code, and error.
// The standard http.Client is used unless an optional one is supplied in the client parameter.
func (t *Tools) PostJSONToRemote(uri string, data interface{}, client ...*http.Client) (*http.Response, int, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	httpClient := http.Client{}
	if len(client) > 0 {
		httpClient = *client[0]
	}

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewBuffer(payload))
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	defer res.Body.Close()

	return res, res.StatusCode, nil
}

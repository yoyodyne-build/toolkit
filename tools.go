package toolkit

import (
	"crypto/rand"
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
// saving it as teh value provided in the displayName parameter
func (t *Tools) DownloadStaticFile(w http.ResponseWriter, r *http.Request, pathName, fileName, displayName string) {
	fp := filepath.Join(pathName, fileName)

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", displayName))

	http.ServeFile(w, r, fp)
}

// GetNewFileName generates a new file name
func (t *Tools) GetNewFileName(fileHeader *multipart.FileHeader, renameFile bool) string {
	if renameFile {
		return fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(fileHeader.Filename))
	} else {
		return fileHeader.Filename
	}
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

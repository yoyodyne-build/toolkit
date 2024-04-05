package toolkit

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
			uploadedFiles, err = func(uploadedFiles []*UploadedFile) ([]*UploadedFile, error) {
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
				allowed := false
				fileType := http.DetectContentType(buff)

				if len(t.AllowedFileTypes) == 0 {
					t.AllowedFileTypes = []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "application/pdf"}
				}

				if len(t.AllowedFileTypes) > 0 {
					for _, t := range t.AllowedFileTypes {
						if strings.EqualFold(fileType, t) {
							allowed = true
							break
						}
					}
				} else {
					// allowing all file types
					allowed = true
				}

				if !allowed {
					return nil, errors.New("file type not permitted")
				}

				// be kind, rewind
				_, err = infile.Seek(0, 0)
				if err != nil {
					return nil, err
				}

				uploadedFile.OriginalFileName = fileHeader.Filename

				if renameFile {
					uploadedFile.NewFileName = fmt.Sprintf("%s%s", t.RandomString(25), filepath.Ext(fileHeader.Filename))
				} else {
					uploadedFile.NewFileName = fileHeader.Filename
				}

				var outfile *os.File
				defer outfile.Close()

				if outfile, err = os.Create(filepath.Join(uploadDir, uploadedFile.NewFileName)); err != nil {
					return nil, err
				} else {
					fileSize, err := io.Copy(outfile, infile)
					if err != nil {
						return nil, err
					}
					uploadedFile.FileSize = fileSize
				}

				uploadedFiles = append(uploadedFiles, &uploadedFile)

				return uploadedFiles, nil
			}(uploadedFiles)
			if err != nil {
				return uploadedFiles, err
			}
		}
	}

	return uploadedFiles, nil
}

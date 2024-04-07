package toolkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func MockTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestTools_CheckFileType(t *testing.T) {
	var tools Tools

	if tools.CheckFileType("image/jpeg") == false {
		t.Error("CheckFileType returned false for image/jpeg")
	}

	if tools.CheckFileType("image/jpg") == false {
		t.Error("CheckFileType returned false for image/jpg")
	}

	if tools.CheckFileType("image/png") == false {
		t.Error("CheckFileType returned false for image/png")
	}

	if tools.CheckFileType("image/gif") == false {
		t.Error("CheckFileType returned false for image/gif")
	}

	if tools.CheckFileType("application/pdf") == false {
		t.Error("CheckFileType returned false for application/pdf")
	}

	if tools.CheckFileType("text/plain") == true {
		t.Error("CheckFileType returned true for text/plain")
	}
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var tools Tools
	testDir := "./foo/bar/baz"

	err := tools.CreateDirIfNotExist(testDir)
	if err != nil {
		t.Error(err)
	}

	err = tools.CreateDirIfNotExist(testDir)
	if err != nil {
		t.Error(err)
	}

	_ = os.RemoveAll("./foo")
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	var tools Tools

	displayName := "loljohnny.jpg"
	expectedFileSize := "32152"
	expectedDisposition := fmt.Sprintf("attachment; filename=\"%s\"", displayName)

	tools.DownloadStaticFile(rr, req, "./testdata", "tipfinger.jpg", displayName)

	res := rr.Result()
	defer res.Body.Close()

	actualFileSize := res.Header["Content-Length"][0]
	if actualFileSize != expectedFileSize {
		t.Errorf("Incorrect size: got %s expected %s", actualFileSize, expectedFileSize)
	}

	actualDisposition := res.Header["Content-Disposition"][0]
	if actualDisposition != expectedDisposition {
		t.Errorf("Incorrect disposition: got %s expected %s", actualDisposition, expectedDisposition)
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

func TestTools_GetNewFileName(t *testing.T) {
	var tools Tools
	var fileHeader multipart.FileHeader

	fileHeader.Filename = "cyborg-ape.png"

	newFileName := tools.GetNewFileName(&fileHeader, true)

	if len(newFileName) != 29 {
		t.Error("GetNewFileName returned a string of the wrong length")
	}

	newFileName = tools.GetNewFileName(&fileHeader, false)

	if newFileName != "cyborg-ape.png" {
		t.Error("GetNewFileName returned an unexpected file name")
	}
}

func TestTools_RandomString(t *testing.T) {
	var tools Tools

	s := tools.RandomString(10)

	if len(s) != 10 {
		t.Error("RandomString returned a string of the wrong length")
	}
}

var slugifyTests = []struct {
	name          string
	input         string
	expected      string
	errorExpected bool
}{
	{name: "empty string", input: "  ", expected: "", errorExpected: true},
	{name: "garbage string", input: "&+=^", expected: "", errorExpected: true},
	{name: "normal string", input: " hello world ", expected: "hello-world", errorExpected: false},
	{name: "odd but valid string", input: " hello   world ", expected: "hello-world", errorExpected: false},
	{name: "complex string", input: "Hugo, what the *!&~ are YOU UP to Dawg?", expected: "hugo-what-the-are-you-up-to-dawg", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var tools Tools

	for _, entry := range slugifyTests {
		actual, err := tools.Slugify(entry.input)
		if err != nil && !entry.errorExpected {
			t.Errorf("%s: unexpected error: %s", entry.name, err.Error())
		}

		if err == nil && entry.errorExpected {
			t.Errorf("%s: error expected but none received", entry.name)
		}

		if actual != entry.expected {
			t.Errorf("%s: expected '%s', got '%s'", entry.name, entry.expected, actual)
		}
	}
}

var uploadTests = []struct {
	name          string
	allowedTypes  []string
	renameFile    bool
	errorExpected bool
}{
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: false, errorExpected: false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg", "image/png"}, renameFile: true, errorExpected: false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile: false, errorExpected: true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, entry := range uploadTests {
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			part, err := writer.CreateFormFile("file", "./testdata/cyborg-ape.png")
			if err != nil {
				t.Error(err)
				return
			}

			file, err := os.Open("./testdata/cyborg-ape.png")
			if err != nil {
				t.Error("error opening image file", err)
				return
			}
			defer file.Close()

			img, _, err := image.Decode(file)
			if err != nil {
				t.Error("error decoding image", err)
				return
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error("error encoding image", err)
				return
			}

		}()

		request := httptest.NewRequest(http.MethodPost, "/", pr)
		request.Header.Set("Content-Type", writer.FormDataContentType())

		var testTools Tools
		testTools.AllowedFileTypes = entry.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", entry.renameFile)
		if err != nil && !entry.errorExpected {
			t.Error(err)
		}

		if !entry.errorExpected {
			target := fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)
			if _, err := os.Stat(target); os.IsNotExist(err) {
				t.Errorf("%s - expected file to exist: %s", entry.name, err.Error())
			}

			_ = os.Remove(target)
		}

		if !entry.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", entry.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadFile(t *testing.T) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		part, err := writer.CreateFormFile("file", "./testdata/cyborg-ape.png")
		if err != nil {
			t.Error(err)
			return
		}

		file, err := os.Open("./testdata/cyborg-ape.png")
		if err != nil {
			t.Error("error opening image file", err)
			return
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			t.Error("error decoding image", err)
			return
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error("error encoding image", err)
			return
		}

	}()

	request := httptest.NewRequest(http.MethodPost, "/", pr)
	request.Header.Set("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	target := fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)
	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	_ = os.Remove(target)
}

var readJSONTests = []struct {
	name          string
	json          string
	errorExpected bool
	maxSize       int64
	allowUnknown  bool
}{
	{name: "valid JSON", json: `{"name": "John Doe"}`, errorExpected: false, maxSize: 512, allowUnknown: false},
	{name: "invalid JSON", json: `{"name": "John Doe"`, errorExpected: true, maxSize: 512, allowUnknown: false},
	{name: "not JSON", json: "If there was a problem, yo I'll solve it", errorExpected: true, maxSize: 512, allowUnknown: false},
	{name: "invalid type", json: `{"name": 1}`, errorExpected: true, maxSize: 512, allowUnknown: false},
	{name: "missing field name", json: `{name: "Bilbo Biggins"}`, errorExpected: true, maxSize: 512, allowUnknown: true},
	{name: "allowed unknown field", json: `{"name": "Pat McCorkindale", "age": 15}`, errorExpected: false, maxSize: 512, allowUnknown: true},
	{name: "disallowed unknown field", json: `{"name": "Pat McCorkindale", "age": 15}`, errorExpected: true, maxSize: 512, allowUnknown: false},
	{name: "payload exceeds size limit", json: `{"name": "John Doe"}`, errorExpected: true, maxSize: 5, allowUnknown: false},
	{name: "empty payloads", json: "", errorExpected: true, maxSize: 512, allowUnknown: false},
	{name: "multiple payloads", json: `{"name": "John Doe"}{"name": "Jane Doe"}`, errorExpected: true, maxSize: 512, allowUnknown: false},
}

func TestTools_ReadJSON(t *testing.T) {
	var tools Tools
	for _, entry := range readJSONTests {
		tools.MaxJSONSize = entry.maxSize
		tools.AllowUnknownFields = entry.allowUnknown

		var decoded struct {
			Name string `json:"name"`
		}

		req, err := http.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(entry.json)))
		if err != nil {
			t.Log("Error:", err)
		}

		rr := httptest.NewRecorder()
		err = tools.ReadJSON(rr, req, &decoded)

		if err != nil && !entry.errorExpected {
			t.Errorf("%s: unexpected error: %s", entry.name, err.Error())
		}

		if entry.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", entry.name)
		}

		req.Body.Close()
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var tools Tools
	rr := httptest.NewRecorder()
	payload := JSONResponse{
		Error:   false,
		Message: "foo",
	}
	headers := make(http.Header)
	headers.Add("X-Test", "foo")

	err := tools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

func TestTools_ErrorJSON(t *testing.T) {
	var tools Tools
	rr := httptest.NewRecorder()

	errorText := "not to be used for the other use"
	errorStatus := http.StatusServiceUnavailable
	err := tools.ErrorJSON(rr, errors.New(errorText), errorStatus)
	if err != nil {
		t.Error(err)
	}

	var payload JSONResponse
	decoder := json.NewDecoder(rr.Body)
	err = decoder.Decode(&payload)
	if err != nil {
		t.Errorf("error decoding JSON: %v", err)
	}

	if !payload.Error {
		t.Error("expected error to be true")
	}

	if payload.Message != errorText {
		t.Errorf("expected message to be %s, got %s", errorText, payload.Message)
	}

	if rr.Code != errorStatus {
		t.Errorf("expected request status code %d, got %d", errorStatus, rr.Code)
	}
}

func TestTools_PostJSONToRemote(t *testing.T) {
	client := MockTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"status": "ok"}`)),
			Header:     make(http.Header),
		}
	})

	var data struct {
		Bar string `json:"bar"`
	}
	data.Bar = "baz"

	var tools Tools

	_, _, err := tools.PostJSONToRemote("http://example.com", data, client)
	if err != nil {
		t.Errorf("failed to post JSON to remote: %v", err)
	}
}

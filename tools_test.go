package toolkit

import (
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

<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>
[![Version](https://img.shields.io/badge/goversion-1.22.x-blue.svg)](https://golang.org)
[![License](http://img.shields.io/badge/License-BSD_3--Clause-blue.svg?style=flat-square)](https://raw.githubusercontent.com/yoyodyne-build/toolkit/main/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/yoyodyne-build/toolkit)](https://goreportcard.com/report/github.com/yoyodyne-build/toolkit)
<a href="https://pkg.go.dev/github.com/yoyodyne-build/toolkit"><img src="https://img.shields.io/badge/godoc-reference-%23007d9c.svg"></a>
![Tests](https://github.com/yoyodyne-build/toolkit/actions/workflows/tests.yml/badge.svg)
[![Go Coverage](https://github.com/yoyodyne-build/toolkit/wiki/coverage.svg)](https://raw.githack.com/wiki/yoyodyne-build/toolkit/coverage.html)

# Toolkit

Go module with some common utility functions.

Included tools are:

- **CreateDirIfNotExist**: Create a directory, including parent directories, if it does not exist
- **DownloadStaticFile**: Downloads a static file from a given directory
- **JSON tools**:
  - **ReadJSON**: Read JSON
  - **WriteJSON**: Write JSON
  - **ErrorJSON**: Produce a JSON encoded error response
  - **PostJSONToRemote**: Post JSON to a remote service
- **RandomString**: Returns a random string of length _n_
- **Slugify**: Create an URL safe slug from a string
- **UploadFile**: Upload a file to a specified location
- **UploadFiles**: Upload multiple files to a specified location

## Installation

```shell
go get github.com/yoyodyne-build/toolkit
```

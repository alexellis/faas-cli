// Copyright (c) Alex Ellis 2017. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package commands

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openfaas/faas-cli/proxy"
)

const (
	defaultTemplateRepository = "https://github.com/openfaas/faas-cli"
	templateDirectory         = "./template/"
	rootLanguageDirSplitCount = 3
)

type ExtractAction int

const (
	ShouldExtractData ExtractAction = iota
	NewTemplateFound
	DirectoryAlreadyExists
	SkipWritingData
)

// fetchTemplates fetch code templates from GitHub master zip file.
func fetchTemplates(templateURL string, overwrite bool) error {

	if len(templateURL) == 0 {
		templateURL = defaultTemplateRepository
	}

	archive, err := fetchMasterZip(templateURL)
	if err != nil {
		removeArchive(archive)
		return err
	}

	log.Printf("Attempting to expand templates from %s\n", archive)

	preExistingLanguages, fetchedLanguages, err := expandTemplatesFromZip(archive, overwrite)
	if err != nil {
		return err
	}

	if len(preExistingLanguages) > 0 {
		log.Printf("Cannot overwrite the following %d directories: %v\n", len(preExistingLanguages), preExistingLanguages)
	}

	log.Printf("Fetched %d template(s) : %v from %s\n", len(fetchedLanguages), fetchedLanguages, templateURL)

	err = removeArchive(archive)

	return err
}

// expandTemplatesFromZip() takes a path to an archive, and whether or not
// we are allowed to overwrite pre-existing language templates. It returns
// a list of languages that already exist (could not be overwritten), and
// a list of languages that are newly downloaded.
func expandTemplatesFromZip(archive string, overwrite bool) ([]string, []string, error) {
	var existingLanguages []string
	var fetchedLanguages []string
	availableLanguages := make(map[string]bool)

	zipFile, err := zip.OpenReader(archive)
	if err != nil {
		return nil, nil, err
	}

	for _, z := range zipFile.File {
		var rc io.ReadCloser

		relativePath := z.Name[strings.Index(z.Name, "/")+1:]
		if strings.Index(relativePath, "template/") != 0 {
			// Process only directories inside "template" at root
			continue
		}

		action, language, isDirectory := canExpandTemplateData(availableLanguages, relativePath)

		var expandFromZip bool

		switch action {

		case ShouldExtractData:
			expandFromZip = true
		case NewTemplateFound:
			expandFromZip = true
			fetchedLanguages = append(fetchedLanguages, language)
		case DirectoryAlreadyExists:
			expandFromZip = false
			existingLanguages = append(existingLanguages, language)
		case SkipWritingData:
			expandFromZip = false
		default:
			return nil, nil, errors.New(fmt.Sprintf("Don't know what to do when extracting zip: %s", archive))

		}

		if expandFromZip {
			if rc, err = z.Open(); err != nil {
				break
			}

			if err = createPath(relativePath, z.Mode()); err != nil {
				break
			}

			// If relativePath is just a directory, then skip expanding it.
			if len(relativePath) > 1 && !isDirectory {
				if err = writeFile(rc, z.UncompressedSize64, relativePath, z.Mode()); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	zipFile.Close()
	return existingLanguages, fetchedLanguages, nil
}

// canExpandTemplateData() takes the map of available languages, and the
// path to a file in the zip archive. Returns what we should do with the file
// in form of ExtractAction enum, the language name, and whether it is a directory
func canExpandTemplateData(availableLanguages map[string]bool, relativePath string) (ExtractAction, string, bool) {
	if pathSplit := strings.Split(relativePath, "/"); len(pathSplit) > 2 {
		language := pathSplit[1]

		// We know that this path is a directory if the last character is a "/"
		isDirectory := relativePath[len(relativePath)-1:] == "/"

		// Check if this is the root directory for a language (at ./template/lang)
		if len(pathSplit) == rootLanguageDirSplitCount && isDirectory {
			if !canWriteLanguage(availableLanguages, language, overwrite) {
				return DirectoryAlreadyExists, language, isDirectory
			}
			return NewTemplateFound, language, isDirectory
		} else {
			if !canWriteLanguage(availableLanguages, language, overwrite) {
				return SkipWritingData, language, isDirectory
			}
			return ShouldExtractData, language, isDirectory
		}
	}
	// template/
	return SkipWritingData, "", true
}

// removeArchive removes the given file
func removeArchive(archive string) error {
	log.Printf("Cleaning up zip file...")
	if _, err := os.Stat(archive); err == nil {
		return os.Remove(archive)
	} else {
		return err
	}
}

// fetchMasterZip downloads a zip file from a repository URL
func fetchMasterZip(templateURL string) (string, error) {
	var err error

	templateURL = strings.TrimRight(templateURL, "/")
	templateURL = templateURL + "/archive/master.zip"
	archive := "master.zip"

	if _, err := os.Stat(archive); err != nil {
		timeout := 120 * time.Second
		client := proxy.MakeHTTPClient(&timeout)

		req, err := http.NewRequest(http.MethodGet, templateURL, nil)
		if err != nil {
			log.Println(err.Error())
			return "", err
		}
		log.Printf("HTTP GET %s\n", templateURL)
		res, err := client.Do(req)
		if err != nil {
			log.Println(err.Error())
			return "", err
		}
		if res.StatusCode != http.StatusOK {
			err := errors.New(fmt.Sprintf("%s is not valid, status code %d", templateURL, res.StatusCode))
			log.Println(err.Error())
			return "", err
		}
		if res.Body != nil {
			defer res.Body.Close()
		}
		bytesOut, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println(err.Error())
			return "", err
		}

		log.Printf("Writing %dKb to %s\n", len(bytesOut)/1024, archive)
		err = ioutil.WriteFile(archive, bytesOut, 0700)
		if err != nil {
			log.Println(err.Error())
			return "", err
		}
	}
	fmt.Println("")
	return archive, err
}

func writeFile(rc io.ReadCloser, size uint64, relativePath string, perms os.FileMode) error {
	var err error

	defer rc.Close()
	f, err := os.OpenFile(relativePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perms)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.CopyN(f, rc, int64(size))

	return err
}

func createPath(relativePath string, perms os.FileMode) error {
	dir := filepath.Dir(relativePath)
	err := os.MkdirAll(dir, perms)
	return err
}

// canWriteLanguage() tells whether the language can be expanded from the zip or not.
// availableLanguages map keeps track of which languages we know to be okay to copy.
// overwrite flag will allow to force copy the language template
func canWriteLanguage(availableLanguages map[string]bool, language string, overwrite bool) bool {
	canWrite := false
	if availableLanguages != nil && len(language) > 0 {
		if _, found := availableLanguages[language]; found {
			return availableLanguages[language]
		}
		canWrite = templateFolderExists(language, overwrite)
		availableLanguages[language] = canWrite
	}

	return canWrite
}

// Takes a language input (e.g. "node"), tells whether or not it is OK to download
func templateFolderExists(language string, overwrite bool) bool {
	dir := templateDirectory + language
	if _, err := os.Stat(dir); err == nil && !overwrite {
		// The directory template/language/ exists
		return false
	}
	return true
}

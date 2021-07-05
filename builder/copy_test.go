package builder

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func Test_CopyFiles(t *testing.T) {
	fileModes := []int{0600, 0640, 0644, 0700, 0755}

	dir := os.TempDir()
	for _, mode := range fileModes {
		// set up a source folder with 2 file
		srcDir, srcDirErr := setupSourceFolder(2, mode)
		if srcDirErr != nil {
			log.Fatal("Error creating source folder")
		}
		defer os.RemoveAll(srcDir)

		// create a destination folder to copy the files to
		destDir, destDirErr := ioutil.TempDir(dir, "openfaas-test-destination-")
		if destDirErr != nil {
			t.Fatalf("Error creating destination folder\n%v", destDirErr)
		}
		defer os.RemoveAll(destDir)

		err := CopyFiles(srcDir, destDir+"/")
		if err != nil {
			t.Fatalf("Unexpected copy error\n%v", err)
		}

		err = checkDestinationFiles(destDir, 2, mode, nil)
		if err != nil {
			t.Fatalf("Destination file mode differs from source file mode\n%v", err)
		}
	}
}

func Test_CopyFiles_ToDestinationWithIntermediateFolder(t *testing.T) {
	dir := os.TempDir()
	data := []byte("open faas")

	// create a folder for source files
	srcDir, dirError := ioutil.TempDir(dir, "openfaas-test-source-")
	if dirError != nil {
		t.Fatalf("Error creating source folder\n%v", dirError)
	}
	defer os.RemoveAll(srcDir)

	// create a file inside the created folder
	mode := 0600
	srcFile := fmt.Sprintf("%s/test-file-1", srcDir)
	fileErr := ioutil.WriteFile(srcFile, data, os.FileMode(mode))
	if fileErr != nil {
		t.Fatalf("Error creating source file\n%v", dirError)
	}

	// create a destination folder to copy the files to
	destDir, destDirErr := ioutil.TempDir(dir, "openfaas-test-destination-")
	if destDirErr != nil {
		t.Fatalf("Error creating destination folder\n%v", destDirErr)
	}
	defer os.RemoveAll(destDir)

	err := CopyFiles(srcFile, destDir+"/intermediate/test-file-1")
	if err != nil {
		t.Fatalf("Unexpected copy error\n%v", err)
	}

	err = checkDestinationFiles(destDir+"/intermediate/", 1, mode, nil)
	if err != nil {
		t.Fatalf("Destination file mode differs from source file mode\n%v", err)
	}
}

func Test_CopyFilesWithIgnorePaths(t *testing.T) {
	fileModes := []int{0600, 0640, 0644, 0700, 0755}

	dir := os.TempDir()
	for _, mode := range fileModes {
		// set up a source folder with 2 file
		srcDir, srcDirErr := setupSourceFolder(2, mode)
		if srcDirErr != nil {
			log.Fatal("Error creating source folder")
		}
		defer os.RemoveAll(srcDir)

		// create a destination folder to copy the files to
		destDir, destDirErr := ioutil.TempDir(dir, "openfaas-test-destination-")
		if destDirErr != nil {
			t.Fatalf("Error creating destination folder\n%v", destDirErr)
		}
		defer os.RemoveAll(destDir)

		ignorePaths := []string{
			fmt.Sprintf("%s/test-file-2", destDir),
		}

		err := CopyFilesWithIgnorePaths(srcDir, destDir+"/", ignorePaths)
		if err != nil {
			t.Fatalf("Unexpected copy error\n%v", err)
		}

		err = checkDestinationFiles(destDir, 2, mode, ignorePaths)
		if err != nil {
			t.Fatalf("Destination file mode differs from source file mode\n%v", err)
		}
	}
}

func setupSourceFolder(numberOfFiles, mode int) (string, error) {
	dir := os.TempDir()
	data := []byte("open faas")

	// create a folder for source files
	srcDir, dirError := ioutil.TempDir(dir, "openfaas-test-source-")
	if dirError != nil {
		return "", dirError
	}

	// create n files inside the created folder
	for i := 1; i <= numberOfFiles; i++ {
		srcFile := fmt.Sprintf("%s/test-file-%d", srcDir, i)
		fileErr := ioutil.WriteFile(srcFile, data, os.FileMode(mode))
		if fileErr != nil {
			return "", fileErr
		}
	}

	return srcDir, nil
}

func checkDestinationFiles(dir string, numberOfFiles, mode int, ignorePaths []string) error {
	// Check each file inside the destination folder
	for i := 1; i <= numberOfFiles; i++ {
		filePath := fmt.Sprintf("%s/test-file-%d", dir, i)
		fileStat, err := os.Stat(filePath)

		for _, ignorePath := range ignorePaths {
			if filePath == ignorePath {
				if !os.IsNotExist(err) {
					return err
				}
			} else {
				if os.IsNotExist(err) {
					return err
				}

				if fileStat.Mode() != os.FileMode(mode) {
					return errors.New("expected mode did not match")
				}
			}
		}
	}

	return nil
}

package builder

// Copy "recursivelies copy a file object from source to dest while perserving
import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// CopyFiles copies files from src to destination.
func CopyFiles(src, dest string) error {
	return CopyFilesWithIgnorePaths(src, dest, nil)
}

// CopyFilesWithIgnorePaths copies files from src to destination,
// but ignores files or directories specified by ignorePaths
func CopyFilesWithIgnorePaths(src, dest string, ignorePaths []string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	for _, ignorePath := range ignorePaths {
		if dest == ignorePath {
			return nil
		}
	}

	if info.IsDir() {
		debugPrint(fmt.Sprintf("Creating directory: %s at %s", info.Name(), dest))
		return copyDir(src, dest, ignorePaths)
	}

	debugPrint(fmt.Sprintf("cp - %s %s", src, dest))
	return copyFile(src, dest)
}

// copyDir will recursively copy a directory to dest
func copyDir(src, dest string, ignorePaths []string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error reading dest stats: %s", err.Error())
	}

	if err := os.MkdirAll(dest, info.Mode()); err != nil {
		return fmt.Errorf("error creating path: %s - %s", dest, err.Error())
	}

	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, info := range infos {
		if err := CopyFilesWithIgnorePaths(
			filepath.Join(src, info.Name()),
			filepath.Join(dest, info.Name()),
			ignorePaths,
		); err != nil {
			return err
		}
	}

	return nil
}

// copyFile will copy a file with the same mode as the src file
func copyFile(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("error reading src file stats: %s", err.Error())
	}

	err = ensureBaseDir(dest)
	if err != nil {
		return fmt.Errorf("error creating dest base directory: %s", err.Error())
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("error creating dest file: %s", err.Error())
	}
	defer f.Close()

	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
		return fmt.Errorf("error setting dest file mode: %s", err.Error())
	}

	s, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening src file: %s", err.Error())
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	if err != nil {
		return fmt.Errorf("Error copying dest file: %s\n" + err.Error())
	}

	return nil
}

// ensureBaseDir creates the base directory of the given file path, if needed.
// For example, if fpath is 'build/extras/dictionary.txt`, ensureBaseDir will
// make sure that the directory `buid/extras/` is created.
func ensureBaseDir(fpath string) error {
	baseDir := path.Dir(fpath)
	info, err := os.Stat(baseDir)
	if err == nil && info.IsDir() {
		return nil
	}
	return os.MkdirAll(baseDir, 0755)
}

func debugPrint(message string) {

	if val, exists := os.LookupEnv("debug"); exists && (val == "1" || val == "true") {
		fmt.Println(message)
	}
}

package common

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.

	if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
		const dirPerm = 0700
		if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
			return err
		}
	}

	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

func getFileList(root, expZipFile string, isExtName bool) ([]string, error) {

	files, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	fileList := make([]string, 0, len(files))
	for _, file := range files {
		// xxxx.pa结尾为有效文件
		if isExtName {
			fileTemp := strings.Split(file.Name(), ".")
			if fileTemp[len(fileTemp)-1] == expZipFile {
				fileList = append(fileList, file.Name())
			}
		} else {
			fileTemp := strings.Split(file.Name(), ".")
			if fileTemp[len(fileTemp)-1] == expZipFile {
				fileList = append(fileList, fileTemp[0])
			}
		}
	}
	return fileList, nil
}

func readKeyFile(file string) ([]byte, error) {
	jsonBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func getTimestamp() string {
	ts := time.Now().UTC()
	return fmt.Sprintf("UTC-%s", toISO8601(ts))
}

func toISO8601(t time.Time) string {
	var tz string
	name, offset := t.Zone()
	if name == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d-%02d-%02d.%09d%s", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), tz)
}

func joinPath(root, filename string) string {
	if filepath.IsAbs(filename) {
		return filename
	} else {
		return filepath.Join(root, filename)
	}
}

func checkPassword(passStr string) error {

	passphrase := []byte(passStr)

	if len(passphrase) < minPasswordLength {
		return ErrInvalidPasswordPolicy
	}

	var isUpper, isLower, isNumber bool
	for _, c := range passphrase {
		if c >= 'a' && c <= 'z' {
			isLower = true
		}

		if c >= 'A' && c <= 'Z' {
			isUpper = true
		}

		if c >= '0' && c <= '9' {
			isNumber = true
		}
	}

	if isNumber && isUpper && isLower {
		return nil
	}
	return ErrInvalidPasswordPolicy
}

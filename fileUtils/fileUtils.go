package fileUtils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Location int // Define a custom type for clarity and type safety

const (
	INPUT Location = iota
	OUTPUT
	TEMP
	CACHE
	FIXES
)

const (
	INPUT_DATA_LOCATION  = "input-data"
	OUTPUT_DATA_LOCATION = "gtfs"
	TEMP_DATA_LOCATION   = "temp-data"
	CACHE_DATA_LOCATION  = "cache-data"
	FIXES_DATA_LOCATION  = "fixes-data"
)

var locations = map[Location]string{
	INPUT:  INPUT_DATA_LOCATION + "/%s",
	OUTPUT: OUTPUT_DATA_LOCATION + "/%s",
	TEMP:   TEMP_DATA_LOCATION + "/%s",
	CACHE:  CACHE_DATA_LOCATION + "/%s",
	FIXES:  FIXES_DATA_LOCATION + "/%s",
}

// LoadFile Loads data from a file.
// file_name The name of the file to load data from.
// loc The location where the file should be loaded from (INPUT, OUTPUT, TEMP, or CACHE).
// :return: The data loaded from the file.
func LoadFile(fileName string, loc Location) ([]byte, error) {
	jsonFile, err := os.Open(getFilePath(fileName, loc))
	if err != nil {
		fmt.Println(err)
		return []byte{}, err
	}
	// fmt.Println("Successfully opened file ", fileName)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()
	fileBytes, _ := io.ReadAll(jsonFile)
	return fileBytes, nil
}

func getFilePath(fileName string, loc Location) string {
	return fmt.Sprintf(locations[loc], fileName)
}

func SaveFile(fileName string, data []byte, loc Location) error {
	// Ensure the directory exists
	dirs := ""
	if strings.Contains(fileName, "/") {
		dirs = fileName[:strings.LastIndex(fileName, "/")]
	}
	err := os.MkdirAll(getFilePath(dirs, loc), os.ModePerm)
	if err != nil {
		return err
	}
	// Write the data to the file
	return os.WriteFile(getFilePath(fileName, loc), data, 0644)
}

func (loc Location) String() string {
	return strings.Trim(locations[loc], "/%s")
}

func (loc Location) GetZipString() string {
	return loc.String() + ".zip"
}

func (loc Location) Zip() error {
	zipFile, err := os.Create(loc.GetZipString())
	if err != nil {
		panic(err)
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Ensure that `path` is not absolute; it should not start with "/".
		// This snippet happens to work because I don't use
		// absolute paths, but ensure your real-world code
		// transforms path into a zip-root relative path.
		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	err = filepath.Walk(loc.String(), walker)
	if err != nil {
		return err
	}
	return nil

}

func (loc Location) UnZip() error {
	r, err := zip.OpenReader(loc.GetZipString())
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	err = os.MkdirAll(loc.String(), os.ModePerm)
	if err != nil {
		return err
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := f.Name

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, os.ModePerm)
		} else {
			os.MkdirAll(filepath.Dir(path), os.ModePerm)
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func (loc Location) ZipAndClear() error {
	err := loc.Zip()
	if err != nil {
		return err
	}
	return loc.Clear()
}

func (loc Location) UnZipAndClear() error {
	err := loc.UnZip()
	if err != nil {
		return err
	}
	return loc.ClearZip()
}

func (loc Location) Clear() error {
	return os.RemoveAll(loc.String() + "/")
}

func (loc Location) ClearZip() error {
	return os.RemoveAll(loc.GetZipString())
}

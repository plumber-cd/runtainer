package utils

import (
	"github.com/spf13/afero"
)

// OsFs is an instance of afero.NewOsFs
var OsFs = afero.Afero{Fs: afero.NewOsFs()}

// FileExists afero for some reason does not have such a function, so...
func FileExists(filename string) (bool, error) {
	e, err := OsFs.Exists(filename)
	if err != nil {
		return e, err
	}

	e, err = OsFs.IsDir(filename)
	if err != nil {
		return e, err
	}

	return !e, nil
}

package file

import "github.com/spf13/afero"

func SetFS(newFS afero.Fs) {
	fs = newFS
}

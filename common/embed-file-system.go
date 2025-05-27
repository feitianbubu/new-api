package common

import (
	"embed"
	"github.com/gin-contrib/static"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// Credit: https://github.com/gin-contrib/static/issues/19

type embedFileSystem struct {
	http.FileSystem
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	_, err := e.Open(path)
	if err != nil {
		return false
	}
	return true
}

type devFileSystem struct {
	http.FileSystem
	root string
}

func (d devFileSystem) Exists(_ string, path string) bool {
	fullPath := filepath.Join(d.root, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

func EmbedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	if DebugEnabled {
		if _, err := os.Stat(targetPath); err == nil {
			return devFileSystem{
				FileSystem: http.Dir(targetPath),
				root:       targetPath,
			}
		}
	}

	efs, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return embedFileSystem{
		FileSystem: http.FS(efs),
	}
}

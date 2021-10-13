package main

import (
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	ygopro_data "github.com/iamipanda/ygopro-data"
	ygopro_deck_identifier "identifier/ygopro-deck-identifier"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func ensureDirectory(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.Mkdir(path, 0644)
		return "", err
	}

	return path, nil
}

func getSavedCommitId(saveDir string) (string, error) {
	lastCommitIdPath := filepath.Join(saveDir, ".last-commit")
	if _, err := os.Stat(lastCommitIdPath); os.IsNotExist(err) {
		return "", err
	}

	buffer, err := ioutil.ReadFile(lastCommitIdPath)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}

func getLastCommitIdOfRepository(owner string, repo string) string {
	repoDir := filepath.Join(os.TempDir(), owner+"-"+repo)
	err := os.RemoveAll(repoDir)
	if err != nil {
		return ""
	}

	r, err := git.PlainClone(repoDir, false, &git.CloneOptions{
		URL:   "https://github.com/" + owner + "/" + repo,
		Depth: 1,
	})
	if err != nil {
		panic(r)
	}

	ref, err := r.Head()
	if err != nil {
		panic(r)
	}

	return ref.Hash().String()
}

func checkIfUpdatable(owner string, repo string, saveDir string) bool {
	savedLastCommitId, err := getSavedCommitId(saveDir)
	if err != nil {
		return true
	}

	remoteLastCommitId := getLastCommitIdOfRepository(owner, repo)
	if savedLastCommitId != remoteLastCommitId {
		return true
	}

	return false
}

func doUpdate(owner string, repo string, saveDir string, fileFilter func(path string) bool) {
	fs := memfs.New()
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:      "https://github.com/" + owner + "/" + repo,
		Depth:    1,
		Progress: os.Stdout,
	})
	if err != nil {
		panic(err)
	}

	ref, err := r.Head()
	if err != nil {
		panic(err)
	}

	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		panic(err)
	}

	tree, err := commit.Tree()
	if err != nil {
		panic(err)
	}

	err = tree.Files().ForEach(func(f *object.File) error {
		if fileFilter(f.Name) {
			file, err := fs.Open(f.Name)
			if err != nil {
				return err
			}

			buffer := make([]byte, f.Size)
			_, err = file.Read(buffer)
			if err != nil {
				return err
			}

			savePath := filepath.Join(saveDir, filepath.Base(f.Name))
			_, err = ensureDirectory(saveDir)
			if err != nil {
				return err
			}

			err = ioutil.WriteFile(savePath, buffer, 0644)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(filepath.Join(saveDir, ".last-commit"), []byte(ref.Hash().String()), 0644)
	if err != nil {
		panic(err)
	}
}

func main() {
	ygopro_data.LuaPath = filepath.Join(os.Getenv("GOPATH"), "pkg/mod/github.com/iamipanda/ygopro-data@v0.0.0-20190116110429-360968dc5c66/Constant.lua")

	dbOwner := os.Getenv("DATABASE_OWNER")
	dbRepo := os.Getenv("DATABASE_REPO")
	if checkIfUpdatable(dbOwner, dbRepo, "./zh-CN") {
		doUpdate(dbOwner, dbRepo, "./zh-CN", func(path string) bool {
			return strings.HasPrefix(path, "locales/zh-CN/")
		})
	}

	defOwner := os.Getenv("DEFINITION_OWNER")
	defRepo := os.Getenv("DEFINITION_REPO")
	if checkIfUpdatable(defOwner, defRepo, "./ygopro-deck-identifier/Definitions/production") {
		doUpdate(defOwner, defRepo, "./ygopro-deck-identifier/Definitions/production", func(path string) bool {
			return strings.HasSuffix(path, ".deckdef")
		})
	}

	ygopro_deck_identifier.Initialize()
	ygopro_deck_identifier.StartServer()
}

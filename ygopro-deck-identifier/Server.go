package ygopro_deck_identifier

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	ygopro_data "github.com/iamipanda/ygopro-data"
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

func getLastCommitIdOfRepository(owner string, repo string, branch string) string {
	repoDir := filepath.Join(os.TempDir(), owner+"-"+repo)
	err := os.RemoveAll(repoDir)
	if err != nil {
		return ""
	}

	r, err := git.PlainClone(repoDir, false, &git.CloneOptions{
		URL:           "https://github.com/" + owner + "/" + repo,
		Depth:         1,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
		SingleBranch:  true,
	})
	if err != nil {
		panic(err)
	}

	ref, err := r.Head()
	if err != nil {
		panic(err)
	}

	return ref.Hash().String()
}

func checkIfUpdatable(owner string, repo string, branch string, saveDir string) bool {
	savedLastCommitId, err := getSavedCommitId(saveDir)
	if err != nil {
		return true
	}

	remoteLastCommitId := getLastCommitIdOfRepository(owner, repo, branch)
	if savedLastCommitId != remoteLastCommitId {
		return true
	}

	return false
}

func doUpdate(owner string, repo string, branch string, saveDir string, fileFilter func(path string) bool) {
	fs := memfs.New()
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:           "https://github.com/" + owner + "/" + repo,
		Depth:         1,
		Progress:      os.Stdout,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch)),
		SingleBranch:  true,
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

func DoUpdateEnvironment() {
	Logger.Noticef("Check if there are updates for databases and definitions.")

	updated := false

	dbOwner := os.Getenv("DATABASE_OWNER")
	dbRepo := os.Getenv("DATABASE_REPO")
	dbBranch := os.Getenv("DATABASE_BRANCH")
	if checkIfUpdatable(dbOwner, dbRepo, dbBranch, "./zh-CN") {
		Logger.Noticef("Found updates for database.")

		doUpdate(dbOwner, dbRepo, dbBranch, "./zh-CN", func(path string) bool {
			return strings.HasPrefix(path, "locales/zh-CN/")
		})

		Logger.Noticef("Successfully installed updates for database.")
		updated = true
	}

	defOwner := os.Getenv("DEFINITION_OWNER")
	defRepo := os.Getenv("DEFINITION_REPO")
	defBranch := os.Getenv("DEFINITION_BRANCH")
	if checkIfUpdatable(defOwner, defRepo, defBranch, "./ygopro-deck-identifier/Definitions/production") {
		Logger.Noticef("Found updates for identifier definitions.")

		doUpdate(defOwner, defRepo, defBranch, "./ygopro-deck-identifier/Definitions/production", func(path string) bool {
			return strings.HasSuffix(path, ".deckdef")
		})

		Logger.Noticef("Successfully installed updates for identifier definitions.")
		updated = true
	}

	if ApplicationInitialized {
		if updated {
			Logger.Noticef("Now apply updates to the identifier itself.")

			ReloadDatabase()
			ReloadIdentifier("production")

			Logger.Noticef("Successfully applied to the identifier itself.")
		} else {
			Logger.Noticef("There was no updates. Wait for next check ...")
		}
	}
}

func ReloadDatabase() {
	Logger.Info("Reloading database.")
	ygopro_data.LoadAllEnvironmentCards()
	ReloadAllIdentifier()
}

func ReloadIdentifier(name string) {
	GlobalIdentifierMap[name].Reload()
}

func StartServer() {
	router := gin.New()
	router.Use(gin.Recovery())
	if gin.IsDebugging() {
		router.Use(gin.Logger())
	}

	router.POST("/update", func(context *gin.Context) {
		go DoUpdateEnvironment()
		context.String(200, "Ok")
	})

	// pull the database and reset the world.
	router.PATCH("/reload", accessCheck(), func(context *gin.Context) {
		Logger.Info("Reloading database.")
		ygopro_data.LoadAllEnvironmentCards()
		_, text := ReloadAllIdentifier()
		context.String(200, text)
	})

	router.Use(identifierCheck())
	router.POST("/:identifierName", extractDeck(), func(context *gin.Context) {
		identifier := context.MustGet("Identifier").(*IdentifierWrapper)
		deck := context.MustGet("Deck").(ygopro_data.Deck)
		context.JSON(200, identifier.RecognizeAsJson(deck))
	})
	router.POST("/:identifierName/recognize", extractDeck(), func(context *gin.Context) {
		identifier := context.MustGet("Identifier").(*IdentifierWrapper)
		deck := context.MustGet("Deck").(ygopro_data.Deck)
		context.JSON(200, identifier.RecognizeAsJson(deck))
	})

	// 以下的操作，全部需要 Access Key 操作。
	router.Use(accessCheck())

	// 重读数据
	router.POST("/:identifierName/reload", func(context *gin.Context) {
		identifier := context.MustGet("Identifier").(*IdentifierWrapper)
		_, text := identifier.Reload()
		context.String(200, text)
	})
	// 预览数据
	router.POST("/:identifierName/preview", func(context *gin.Context) {
		bytes, _ := context.GetRawData()
		content := string(bytes)
		identifier := context.MustGet("Identifier").(*IdentifierWrapper)
		_, log := identifier.GetCompilePreview(content, "compile")
		context.String(200, log)
	})
	router.POST("/:identifierName/verbose", extractDeck(), func(context *gin.Context) {
		identifier := context.MustGet("Identifier").(*IdentifierWrapper)
		deck := context.MustGet("Deck").(ygopro_data.Deck)
		context.JSON(200, identifier.VerboseRecognizeAsJson(deck))
	})

	// 对运行中的结构，进行读取。
	runtimeApi := router.Group("/:identifierName/runtime")
	{
		runtimeApi.GET("/", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			class := context.Query("class")
			name := context.Query("name")
			if result, ok := identifier.GetRuntimeStructure(class, name); ok {
				context.JSON(200, result)
			} else {
				context.AbortWithStatus(404)
			}
		})
		runtimeApi.GET("/list", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			context.JSON(200, identifier.GetRuntimeList())
		})
		runtimeApi.GET("/deck/:deckName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			deckName := context.Param("deckName")
			if result, ok := identifier.GetRuntimeStructure("deck", deckName); ok {
				context.JSON(200, result)
			} else {
				context.AbortWithStatusJSON(404, "Can't find deck named "+deckName)
			}
		})
		runtimeApi.GET("/tag/:tagName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			tagName := context.Param("tagName")
			if result, ok := identifier.GetRuntimeStructure("tag", tagName); ok {
				context.JSON(200, result)
			} else {
				context.AbortWithStatusJSON(404, "Can't find deck named "+tagName)
			}
		})
		runtimeApi.GET("/set/:setName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			setName := context.Param("setName")
			if result, ok := identifier.GetRuntimeStructure("set", setName); ok {
				context.JSON(200, result)
			} else {
				context.AbortWithStatusJSON(404, "Can't find set named "+setName)
			}
		})
	}

	// 对文件进行操作。
	fileApi := router.Group("/:identifierName/file")
	{
		fileApi.GET("/list", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			list := identifier.GetFileList()
			if list == nil {
				context.AbortWithStatus(500)
			} else {
				context.JSON(200, list)
			}
		})
		fileApi.GET("/single/:fileName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			fileName := context.Param("fileName")
			if content, ok := identifier.GetFile(fileName); ok {
				context.String(200, content)
			} else {
				context.String(404, content)
			}
		})
		fileApi.POST("/pull", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			if content, ok := identifier.Pull(); ok {
				context.String(200, content)
			} else {
				context.String(500, content)
			}
		})
		fileApi.POST("/push", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			bytes, _ := context.GetRawData()
			message := string(bytes)
			if content, ok := identifier.Push(message); ok {
				context.String(200, content)
			} else {
				context.String(500, content)
			}
		})
		fileApi.PUT("/:fileName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			fileName := context.Param("fileName")
			bytes, _ := context.GetRawData()
			content := string(bytes)
			if response, ok := identifier.SetFile(fileName, content); ok {
				context.String(200, content)
			} else {
				context.String(500, response)
			}
		})
	}

	router.Run(Config.Listening)
}

func identifierCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		identifierName := c.Param("identifierName")
		if len(identifierName) == 0 {
			c.AbortWithStatusJSON(404, "You didn't figure the identifier name. Or there is no api like that.")
			return
		}
		if identifier, ok := GlobalIdentifierMap[identifierName]; ok {
			c.Set("Identifier", identifier)
			c.Next()
		} else {
			c.AbortWithStatusJSON(404, "Can't find Identifier named "+identifierName)
		}

	}
}

func accessCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessKey := c.Query("accessKey")
		if len(accessKey) == 0 {
			accessKey = c.PostForm("accessKey")
		}
		if accessKey != Config.AccessKey {
			c.AbortWithStatus(401)
		} else {
			c.Next()
		}
	}
}

func extractDeck() gin.HandlerFunc {
	return func(c *gin.Context) {
		separate_string := c.DefaultQuery("separate", "false")
		separate := separate_string == "true"
		deck := c.PostForm("deck")
		if len(deck) > 0 {
			setDeck(c, deck, separate)
			return
		}
		deck = c.Query("deck")
		if len(deck) > 0 {
			setDeck(c, deck, separate)
			return
		}
		if gin.Mode() == gin.DebugMode {
			buf := make([]byte, 10240)
			num, _ := c.Request.Body.Read(buf)
			deck := string(buf[0:num])
			setDeck(c, deck, separate)
			return
		}
	}
}

func setDeck(c *gin.Context, deckString string, separate bool) {
	deck := ygopro_data.LoadYdkFromString(deckString)
	deck.Summary()
	if separate || gin.Mode() == gin.DebugMode {
		deck.SeparateExFromMain(ygopro_data.GetEnvironment("zh-CN"))
	}
	deck.Classify()
	c.Set("Deck", deck)
}

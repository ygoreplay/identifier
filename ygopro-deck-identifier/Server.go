package ygopro_deck_identifier

import (
	"github.com/gin-gonic/gin"
	"github.com/iamipanda/ygopro-data"
)

func StartServer() {
	router := gin.New()
	router.Use(gin.Recovery())
	if gin.IsDebugging() {
		router.Use(gin.Logger())
	}
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
				context.AbortWithStatusJSON(404, "Can't find deck named " + deckName)
			}
		})
		runtimeApi.GET("/tag/:tagName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			tagName := context.Param("tagName")
			if result, ok := identifier.GetRuntimeStructure("tag", tagName); ok {
				context.JSON(200, result)
			} else {
				context.AbortWithStatusJSON(404, "Can't find deck named " + tagName)
			}
		})
		runtimeApi.GET("/set/:setName", func(context *gin.Context) {
			identifier := context.MustGet("Identifier").(*IdentifierWrapper)
			setName := context.Param("setName")
			if result, ok := identifier.GetRuntimeStructure("set", setName); ok {
				context.JSON(200, result)
			} else {
				context.AbortWithStatusJSON(404, "Can't find set named " + setName)
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
			c.AbortWithStatusJSON(404, "Can't find Identifier named " + identifierName)
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
		deck := c.Query("deck")
		if len(deck) == 0 {
			deck = c.PostForm("deck")
			if len(deck) == 0 {
				c.AbortWithStatus(400)
			} else {
				deck := ygopro_data.LoadYdkFromString(deck)
				deck.Classify()
				c.Set("Deck", deck)
			}
		} else {
			c.AbortWithStatus(501)
		}
	}
}
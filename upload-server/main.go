package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

func main() {
	app := fiber.New()
	app.Use(logger.New())

	const uploadPath = "/cdn-content"

	app.Post("/upload", func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid form data")
		}
		files := form.File["files"]
		if files == nil {
			return fiber.NewError(fiber.StatusBadRequest, "No files uploaded")
		}

		var uploaded []string
		for _, file := range files {
			dst := filepath.Join(uploadPath, file.Filename)
			if err := c.SaveFile(file, dst); err != nil {
				return err
			}
			uploaded = append(uploaded, file.Filename)
		}

		return c.JSON(fiber.Map{"files": uploaded})
	})

	app.Get("/files", func(c *fiber.Ctx) error {
		entries, err := os.ReadDir(uploadPath)
		if err != nil {
			return err
		}
		var files []string
		for _, entry := range entries {
			if !entry.IsDir() {
				files = append(files, entry.Name())
			}
		}
		return c.JSON(files)
	})

	app.Post("/text", func(c *fiber.Ctx) error {
		text := c.FormValue("text")
		if text == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Empty text")
		}

		filename := filepath.Join(uploadPath, "paste_"+generateFilename()+".txt")
		err := os.WriteFile(filename, []byte(text), 0644)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.JSON(fiber.Map{"message": "Text saved", "filename": filepath.Base(filename)})
	})

	app.Listen("0.0.0.0:5000")
}

func generateFilename() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[randSrc.Intn(len(letters))]
	}
	return string(b)
}

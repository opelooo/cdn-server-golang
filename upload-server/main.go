package main

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func generateFilename() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	randSrc := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[randSrc.Intn(len(letters))]
	}
	return string(b)
}

func main() {
	app := fiber.New(fiber.Config{
		BodyLimit: 1000 * 1024 * 1024, // Set a 1000MB body size limit (adjust as needed)
	})
	app.Use(logger.New())

	app.Use(func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("Internal Server Error: %v", r)
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": errMsg,
				})
			}
		}()
		return c.Next()
	})

	const uploadPath = "/cdn-content"
	const textPath = "/cdn-texts"

	// Ensure directories exist
	os.MkdirAll(uploadPath, 0755)
	os.MkdirAll(textPath, 0755)

	// const authToken = "5adda1de909db9153a291dd5f4785c79"
	allowedExtensions := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".pdf":  true,
		".txt":  true,
		".svg":  true,
	}

	// Handle file upload
	app.Post("/upload", func(c *fiber.Ctx) error {
		// // check bearer
		// authHeader := c.Get("Authorization")
		// if authHeader != "Bearer "+authToken {
		// 	return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		// }

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
			// check file type
			ext := filepath.Ext(file.Filename)
			if !allowedExtensions[ext] {
				return fiber.NewError(fiber.StatusBadRequest, "File type not allowed")
			}

			// randomize filename
			dst := filepath.Join(uploadPath, file.Filename)

			if err := c.SaveFile(file, dst); err != nil {
				return err
			}
			uploaded = append(uploaded, file.Filename)
		}

		return c.JSON(fiber.Map{
			"message": "File saved",
			"files":   uploaded,
		})
	})

	// Save text as a file
	app.Post("/text", func(c *fiber.Ctx) error {
		// authHeader := c.Get("Authorization")
		// if authHeader != "Bearer "+authToken {
		// 	return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
		// }

		text := c.FormValue("text")
		if text == "" {
			return fiber.NewError(fiber.StatusBadRequest, "Empty text")
		}

		// Generate a unique filename for the text file
		filename := generateFilename()
		filePath := filepath.Join(uploadPath, filename) // Ensure 'uploadPath' is a valid directory path

		// Save the text to the file
		err := os.WriteFile(filePath, []byte(text), 0644)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		return c.JSON(fiber.Map{
			"message":  "Text saved",
			"filename": filename,
		})
	})

	// Get list of files
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

	// Get list of texts
	app.Get("/texts", func(c *fiber.Ctx) error {
		entries, err := os.ReadDir(textPath)
		if err != nil {
			return err
		}
		var texts []string
		for _, entry := range entries {
			if !entry.IsDir() {
				texts = append(texts, entry.Name())
			}
		}
		return c.JSON(texts)
	})

	// Serve uploaded files and texts as CDN
	app.Get("/cdn/:file", func(c *fiber.Ctx) error {
		file := c.Params("file")

		// Decode the URL-encoded file name
		decodedFile, err := url.QueryUnescape(file)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid file name")
		}

		// Check if it's an actual file (not a text)
		filepath := fmt.Sprintf("%s/%s", uploadPath, decodedFile)
		if _, err := os.Stat(filepath); err == nil {
			return c.SendFile(filepath)
		}

		// Check if it's a text file
		textPath := fmt.Sprintf("%s/%s", textPath, decodedFile)
		if _, err := os.Stat(textPath); err == nil {
			return c.SendFile(textPath)
		}

		return fiber.NewError(fiber.StatusNotFound, "File or text not found")
	})

	// Delete file or text
	app.Delete("/cdn/:file", func(c *fiber.Ctx) error {
		file := c.Params("file")

		// Decode the URL to handle special characters (like spaces %20)
		decodedFile, err := url.QueryUnescape(file)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid file name")
		}

		// Check and delete from file CDN
		filePath := fmt.Sprintf("%s/%s", uploadPath, decodedFile)
		if _, err := os.Stat(filePath); err == nil {
			err := os.Remove(filePath)
			if err != nil {
				return err
			}
			return c.SendString(fmt.Sprintf("File %s deleted", decodedFile))
		}

		// Check and delete from text CDN
		textFilePath := fmt.Sprintf("%s/%s", textPath, decodedFile)
		if _, err := os.Stat(textFilePath); err == nil {
			err := os.Remove(textFilePath)
			if err != nil {
				return err
			}
			return c.SendString(fmt.Sprintf("Text %s deleted", decodedFile))
		}

		return fiber.NewError(fiber.StatusNotFound, "File or text not found")
	})

	app.Listen(":5000")
}

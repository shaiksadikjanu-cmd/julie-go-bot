package main

import (
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

// 1. EMBED THE TEMPLATES FOLDER
// This tells Go to pack the "templates" folder inside the binary file.
//
//go:embed templates/*
var resources embed.FS

// Define the request structure for JSON
type ChatRequest struct {
	Message string `json:"message"`
	Image   string `json:"image"`
}

func main() {
	r := gin.Default()

	// 2. LOAD HTML FROM THE EMBEDDED FILE SYSTEM
	// We use the embedded 'resources' variable instead of looking for a folder on disk.
	tmpl := template.Must(template.New("").ParseFS(resources, "templates/*.html"))
	r.SetHTMLTemplate(tmpl)

	// Serve the UI
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// Handle Chat
	r.POST("/chat", func(c *gin.Context) {
		var req ChatRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Request"})
			return
		}

		ctx := context.Background()
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			// Fallback for local testing
			apiKey = "YOUR_API_KEY_HERE"
		}

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Connection Error"})
			return
		}

		// --- BUILD THE REQUEST ---
		var promptParts []*genai.Part

		if req.Message != "" {
			promptParts = append(promptParts, &genai.Part{Text: req.Message})
		}

		if req.Image != "" {
			b64data := req.Image
			if strings.Contains(b64data, ",") {
				b64data = strings.Split(b64data, ",")[1]
			}
			decoded, err := base64.StdEncoding.DecodeString(b64data)
			if err == nil {
				promptParts = append(promptParts, &genai.Part{
					InlineData: &genai.Blob{
						MIMEType: "image/png",
						Data:     decoded,
					},
				})
			}
		}

		contents := []*genai.Content{ { Parts: promptParts } }

		result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", contents, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI Error: " + err.Error()})
			return
		}

		botText := ""
		if len(result.Candidates) > 0 {
			for _, part := range result.Candidates[0].Content.Parts {
				botText += part.Text
			}
		}
		
		c.JSON(http.StatusOK, gin.H{"response": botText})
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Println("Server running on port " + port)
	r.Run(":" + port)
}
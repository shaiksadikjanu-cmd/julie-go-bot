package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/genai"
)

// Request structure to handle JSON from frontend
type ChatRequest struct {
	Message string `json:"message"`
	Image   string `json:"image"` // Base64 string
}

func main() {
	r := gin.Default()
	
	// Load HTML templates
	r.LoadHTMLGlob("templates/*")
	
	// Serve static files (css, js) if you separate them, 
	// but for now we will put everything in index.html for simplicity.

	// 1. Serve the UI
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 2. Handle Chat (Text + Image)
	r.POST("/chat", func(c *gin.Context) {
		var req ChatRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		ctx := context.Background()
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" { apiKey = "AIzaSyC6OqfaJVHxcPUkZ0S4QMxDJi8CkkXDV_0" }

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Connection Error"})
			return
		}

		// Prepare the parts for Gemini
		var parts []*genai.Part

		// Add Text Part
		if req.Message != "" {
			parts = append(parts, &genai.Part{Text: req.Message})
		}

		// Handle Image if present
		if req.Image != "" {
			// Remove the "data:image/png;base64," prefix if it exists
			b64data := req.Image
			if strings.Contains(b64data, ",") {
				b64data = strings.Split(b64data, ",")[1]
			}
			
			decodedImg, err := base64.StdEncoding.DecodeString(b64data)
			if err == nil {
				// Add image to parts
				parts = append(parts, &genai.Part{
					InlineData: &genai.Blob{
						MIMEType: "image/png", // Assuming PNG/JPEG
						Data:     decodedImg,
					},
				})
			}
		}

		// Wrap parts in a Content object
		contents := []*genai.Content{
			{
				Parts: parts,
			},
		}

		// Call Gemini (Using Flash for speed)
		result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", contents, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI Error: " + err.Error()})
			return
		}

		// Extract clean text
		botText := ""
		if len(result.Candidates) > 0 {
			for _, part := range result.Candidates[0].Content.Parts {
				// Type switch to safely handle Text vs Blob parts in response
				botText += part.Text
			}
		}
		
		c.JSON(http.StatusOK, gin.H{"response": botText})
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Println("Server running on http://localhost:" + port)
	r.Run(":" + port)
}
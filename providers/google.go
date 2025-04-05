package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/elek/rai/config"
	"github.com/elek/rai/schema"
	"github.com/elek/rai/tool"
	"github.com/google/generative-ai-go/genai"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
	"reflect"
	"strings"
)

// GoogleModel implements the schema.Model interface for Google's Gemini API
type GoogleModel struct {
	client *genai.Client
}

// NewGeminiModel creates a new Gemini model client
func NewGeminiModel(p config.Provider) *GoogleModel {

	// Create a new client with the API key
	client, err := genai.NewClient(context.Background(), option.WithAPIKey(p.Key))
	if err != nil {
		// In a real application, we would handle this error better
		// For now, we'll just return a client that will fail on first use
		return &GoogleModel{
			client: nil,
		}
	}

	return &GoogleModel{
		client: client,
	}
}

// Invoke sends a conversation to the Gemini API and returns the response
func (g *GoogleModel) Invoke(ctx context.Context, m config.Model, c *schema.Conversation, tools []schema.Tool) ([]schema.Message, schema.Usage, error) {
	if g.client == nil {
		return nil, schema.Usage{}, errors.New("Gemini client was not initialized properly")
	}

	model := g.client.GenerativeModel(m.Model)

	model.SetMaxOutputTokens(int32(m.MaxToken))
	model.SetTemperature(float32(m.Temperature))

	var contents []*genai.Content
	for _, msg := range c.Messages {
		content := &genai.Content{
			Parts: []genai.Part{
				genai.Text(msg.Content),
			},
		}
		switch msg.Role {
		case "user":
			content.Role = "user"
		case "assistant":
			content.Role = "model"
		default:
			content.Role = "user"
		}

		contents = append(contents, content)
	}

	// Add tools if any
	if len(tools) > 0 {
		var functionDeclarations []*genai.FunctionDeclaration

		for _, t := range tools {
			if t.Callback == nil {
				continue
			}

			funcDecl := &genai.FunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
			}

			sch := &genai.Schema{
				Type:        genai.TypeObject,
				Description: t.Description,
				Properties:  make(map[string]*genai.Schema),
			}
			tool.ProcessParams(t.Callback, func(name string, t reflect.Type, desc string) {
				sch.Properties[name] = &genai.Schema{
					Type:        go2OpenApiType(t),
					Description: desc,
				}
			})
			funcDecl.Parameters = sch
			functionDeclarations = append(functionDeclarations, funcDecl)
		}

		if len(functionDeclarations) > 0 {
			model.Tools = []*genai.Tool{
				{
					FunctionDeclarations: functionDeclarations,
				},
			}
		}
	}

	cs := model.StartChat()
	cs.History = contents[:len(contents)-1]
	resp, err := cs.SendMessage(ctx, contents[len(contents)-1].Parts...)
	if err != nil {
		return nil, schema.Usage{}, errors.WithStack(err)
	}

	if m.Default {
		indent, err := json.MarshalIndent(resp, "", "   ")
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(string(indent))
	}

	// Convert response to schema.Message
	var responseMessages []schema.Message

	for _, candidate := range resp.Candidates {
		if len(candidate.Content.Parts) > 0 {
			// For now, we only handle text responses
			for _, part := range candidate.Content.Parts {
				if textPart, ok := part.(genai.Text); ok {
					responseMessages = append(responseMessages, schema.Message{
						Role:    "assistant",
						Content: string(textPart),
					})
				}
			}
		}
	}

	return responseMessages, schema.Usage{}, nil
}

func go2OpenApiType(t reflect.Type) genai.Type {
	switch t {
	case reflect.TypeOf(""):
		return genai.TypeString
	case reflect.TypeOf(float64(0)):
		return genai.TypeNumber
	case reflect.TypeOf(0):
		return genai.TypeInteger
	case reflect.TypeOf(true):
		return genai.TypeBoolean
	default:
		panic(fmt.Sprintf("unknown type %v", t))
	}
}

func (g *GoogleModel) ListModels(ctx context.Context) ([]schema.ModelVersion, error) {
	models := g.client.ListModels(ctx)
	var resp []schema.ModelVersion
	for {
		next, err := models.Next()
		if err != nil {
			return resp, nil
		}
		_, name, _ := strings.Cut(next.Name, "/")
		resp = append(resp, schema.ModelVersion{
			ID:   "google/" + name,
			Name: next.DisplayName,
		})
	}
}

// Close closes the Gemini client
func (g *GoogleModel) Close() error {
	if g.client != nil {
		return g.client.Close()
	}
	return nil
}

// Ensure GoogleModel implements the Model interface
var _ schema.Model = (*GoogleModel)(nil)

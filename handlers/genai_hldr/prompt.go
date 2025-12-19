package genai_hldr

import (
	_ "embed"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/system_prompt.tmpl
var systemPromptTemplate string

type PromptData struct {
	Now         string
	ChatType    string
	ChatName    string
	BotName     string
	BotUsername string
}

type promptRenderer struct {
	tmpl *template.Template
}

func newPromptRenderer() (*promptRenderer, error) {
	tmpl, err := template.New("system_prompt").Parse(systemPromptTemplate)
	if err != nil {
		return nil, err
	}
	return &promptRenderer{tmpl: tmpl}, nil
}

func (r *promptRenderer) Render(data PromptData) (string, error) {
	var buf strings.Builder
	if data.Now == "" {
		data.Now = time.Now().Format("2006-01-02 15:04:05 -07:00")
	}
	err := r.tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

package formatter

import (
	"fmt"
	"go-progira/internal/domain/types/apitypes"
	"strings"
	"time"
)

func FormatMessageForStackOverflow(updates []apitypes.StackOverFlowUpdate) string {
	content := strings.Builder{}

	for _, update := range updates {
		preview := update.Preview
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		typeText := update.Type.String()
		t := time.Unix(update.CreatedAt, 0)

		text := fmt.Sprintf(
			"Новый %s на StackOverflow\n\n"+
				"Вопрос: %s\n"+
				"Автор: %s\n"+
				"Время: %s\n\n"+
				"Превью:\n%s",
			typeText,
			update.Title,
			update.Owner.DisplayName,
			t,
			preview,
		)

		content.WriteString(text)
	}

	return content.String()
}

func FormatMessageForGithub(updates []apitypes.GithubUpdate) string {
	content := strings.Builder{}

	for _, update := range updates {
		preview := update.Preview
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		typeText := update.Type.String()
		text := fmt.Sprintf(
			"Новый %s на Github\n\n"+
				"Название: %s\n"+
				"Автор: %s\n"+
				"Время: %s\n\n",
			typeText,
			update.Title,
			update.Author.Name,
			update.CreatedAt,
		)

		if preview != "" {
			text += fmt.Sprintf("Превью:\n%s\n\n", preview)
		}

		content.WriteString(text)
	}

	return content.String()
}

package formatter

import (
	"fmt"
	"go-progira/internal/domain/types/api_types"
	"strings"
	"time"
)

func FormatMessageForStackOverflow(updates []api_types.StackOverFlowUpdate) string {
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

func FormatMessageForGithub(updates []api_types.GithubUpdate) string {
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
				"Время: %s\n\n"+
				"Превью:\n%s",
			typeText,
			update.Title,
			update.Author.Name,
			update.CreatedAt,
			preview,
		)
		content.WriteString(text)
	}

	return content.String()
}

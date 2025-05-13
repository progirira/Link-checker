package api_test

import (
	"go-progira/internal/application/scrapper/api"
	"testing"
)

func TestIsStackOverflowURL(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name     string
		given    string
		expected bool
	}

	testCases := []TestCase{
		{
			name:     "url is correct, length is longer than base URL",
			given:    "https://stackoverflow.com/questions/79515510/why-transaction-timeout-in-pgx-doesnt-work",
			expected: true,
		},
		{
			name:     "the length of URL is not long enough",
			given:    "https://stackoverflow.com",
			expected: false,
		},
		{
			name:     "empty url is not Stackoverflow URL",
			given:    "",
			expected: false,
		},
		{
			name:     "not Stackoverflow URL",
			given:    "key",
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			answer := api.IsStackOverflowURL(testCase.given)
			if answer != testCase.expected {
				t.Errorf("Incorrect answer on IsStackOverflowURL, got: %v; expected %v  in case with url %s", answer, testCase.expected, testCase.given)
			}
		})
	}
}

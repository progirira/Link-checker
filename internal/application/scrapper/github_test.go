package scrapper_test

import (
	"errors"
	"go-progira/internal/application/scrapper"
	"go-progira/lib/e"
	"testing"
)

func TestIsGithubURL(t *testing.T) {
	t.Parallel()

	type TestCase struct {
		name     string
		given    string
		expected bool
	}

	testCases := []TestCase{
		{
			name:     "url is correct, length is longer than base URL",
			given:    "https://github.com/todo",
			expected: true,
		},
		{
			name:     "the length of URL is not long enough",
			given:    "https://github",
			expected: false,
		},
		{
			name:     "empty url is not Github URL",
			given:    "",
			expected: false,
		},
		{
			name:     "not Github URL",
			given:    "key",
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			answer := scrapper.IsGitHubURL(testCase.given)
			if answer != testCase.expected {
				t.Errorf("Incorrect answer on isGithubURL, got: %v; expected %v  in case with url %s", answer, testCase.expected, testCase.given)
			}
		})
	}
}

func TestGetOwnerAndRepo(t *testing.T) {
	type TestCase struct {
		name          string
		given         string
		expectedOwner string
		expectedRepo  string
		expectedErr   error
	}

	testCases := []TestCase{
		{
			name:          "url is correct, owner and repo are extracted with no problems",
			given:         "https://github.com/progirira/Golang-projects",
			expectedOwner: "progirira",
			expectedRepo:  "Golang-projects",
			expectedErr:   nil,
		},
		{
			name:          "repo is not specified in URL",
			given:         "https://github.com/progirira",
			expectedOwner: "",
			expectedRepo:  "",
			expectedErr:   e.ErrNoRepoInPath,
		},
		{
			name:          "owner and repo are not specified in URL",
			given:         "https://github.com/",
			expectedOwner: "",
			expectedRepo:  "",
			expectedErr:   e.ErrNoOwnerAndRepoInPath,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			owner, repo, err := scrapper.GetOwnerAndRepo(testCase.given)

			if owner != testCase.expectedOwner {
				t.Errorf("Incorrect owner was extracted, got: %s; expected %s; in case with url %s",
					owner, testCase.expectedOwner, testCase.given)
			}

			if repo != testCase.expectedRepo {
				t.Errorf("Incorrect repository name was extracted, got: %s; expected %s; in case with url %s",
					repo, testCase.expectedRepo, testCase.given)
			}

			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("Incorrect error, got: %v; expected: %v; in case with url %s",
					err, testCase.expectedErr, testCase.given)
			}
		})
	}
}

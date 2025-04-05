package cmd

import (
	"context"
	"github.com/andygrunwald/go-gerrit"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type GerritReview struct {
	Message  string                     `json:"message"`
	Labels   map[string]int             `json:"labels,omitempty"`
	Comments map[string][]GerritComment `json:"comments,omitempty"`
}

// GerritComment represents a single comment in a GerritEnabled review
type GerritComment struct {
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

// postToGerrit converts the review from Claude to GerritEnabled format and posts it
func postToGerrit(ctx context.Context, commit string, lastMsg string, gurl string) error {

	client, err := gerrit.NewClient(ctx, gurl, http.DefaultClient)
	if err != nil {
		return errors.WithStack(err)
	}

	chngs, _, err := client.Changes.QueryChanges(ctx, &gerrit.QueryChangeOptions{
		QueryOptions: gerrit.QueryOptions{
			Query: []string{
				string(commit),
			},
		},
		ChangeOptions: gerrit.ChangeOptions{
			AdditionalFields: []string{
				"ALL_REVISIONS",
			},
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}
	if len(*chngs) == 0 {
		return errors.New("no changes found")
	}
	changeID := (*chngs)[0].ID

	re := regexp.MustCompile("```yaml\n((?s:.*?))\n```")
	matches := re.FindStringSubmatch(lastMsg)
	if len(matches) == 0 {
		return errors.New("no yaml fragment was found found")
	}

	var reviews []ReviewOutput
	err = yaml.Unmarshal([]byte(matches[1]), &reviews)
	if err != nil {
		return errors.WithStack(err)
	}

	comments := map[string][]gerrit.CommentInput{}
	tr := true
	for _, r := range reviews {

		lineNo, err := strconv.Atoi(strings.Split(r.Line, "-")[0])
		if err != nil {
			continue
		}
		if _, found := comments[r.File]; !found {
			comments[r.File] = []gerrit.CommentInput{}
		}
		comments[r.File] = append(comments[r.File], gerrit.CommentInput{
			Line:       lineNo,
			Message:    r.Comment,
			Unresolved: &tr,
		})
	}

	//marshal, err := json.Marshal(comments)
	_, _, err = client.Changes.SetReview(ctx, changeID, commit, &gerrit.ReviewInput{
		Comments: comments,
	})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

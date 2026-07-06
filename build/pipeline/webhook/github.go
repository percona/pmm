package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GitHubClient struct {
	token string
	http  *http.Client
}

func NewGitHubClient(token string) *GitHubClient {
	return &GitHubClient{
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

type PullRequest struct {
	Number int `json:"number"`
	Head   struct {
		SHA  string `json:"sha"`
		Ref  string `json:"ref"`
		Repo struct {
			FullName string `json:"full_name"`
			CloneURL string `json:"clone_url"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

func (c *GitHubClient) GetPR(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", owner, repo, number)
	pr := &PullRequest{}
	if err := c.do(ctx, http.MethodGet, url, nil, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

func (c *GitHubClient) PostComment(ctx context.Context, owner, repo string, issue int, body string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", owner, repo, issue)
	return c.do(ctx, http.MethodPost, url, map[string]string{"body": body}, nil)
}

// ReactToComment adds a reaction (eyes, rocket, +1, -1, laugh, confused,
// heart, hooray) to a PR/issue comment.
func (c *GitHubClient) ReactToComment(ctx context.Context, owner, repo string, commentID int64, content string) error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/comments/%d/reactions", owner, repo, commentID)
	return c.do(ctx, http.MethodPost, url, map[string]string{"content": content}, nil)
}

func (c *GitHubClient) do(ctx context.Context, method, url string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s %s: %s: %s", method, url, resp.Status, bytes.TrimSpace(msg))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

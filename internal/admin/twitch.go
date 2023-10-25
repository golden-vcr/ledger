package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nicklaw5/helix/v2"
)

type ResolveTwitchUserIdFunc func(ctx context.Context, username string) (string, error)

func makeResolveTwitchUserIdFunc(clientId string, clientSecret string) ResolveTwitchUserIdFunc {
	return func(ctx context.Context, username string) (string, error) {
		return resolveTwitchUserId(ctx, clientId, clientSecret, username)
	}
}

func resolveTwitchUserId(ctx context.Context, clientId string, clientSecret string, username string) (string, error) {
	c, err := helix.NewClientWithContext(ctx, &helix.Options{
		ClientID:     clientId,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize Twitch API client: %v", err)
	}

	tokenRes, err := c.RequestAppAccessToken(nil)
	if err == nil && tokenRes.StatusCode != http.StatusOK {
		err = fmt.Errorf("got status %d: %s", tokenRes.StatusCode, tokenRes.ErrorMessage)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get app access token from Twitch API: %w", err)
	}
	c.SetAppAccessToken(tokenRes.Data.AccessToken)

	res, err := c.GetUsers(&helix.UsersParams{
		Logins: []string{username},
	})
	if err == nil && res.StatusCode != http.StatusOK {
		err = fmt.Errorf("got status %d: %s", res.StatusCode, res.ErrorMessage)
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve Twitch user ID from username:%w", err)
	}
	if len(res.Data.Users) != 1 {
		return "", fmt.Errorf("got %d results in response to a single-username lookup", len(res.Data.Users))
	}
	return res.Data.Users[0].ID, nil
}

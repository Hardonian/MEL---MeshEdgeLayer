package config

import (
	"fmt"
	"os"
	"strings"
)

func normalizeIntegration(cfg *Config) {
	if cfg.Integration.MinIntervalSeconds <= 0 {
		cfg.Integration.MinIntervalSeconds = 60
	}
}

func validateIntegration(cfg Config) []string {
	var errs []string
	ic := cfg.Integration
	if !ic.Enabled {
		return errs
	}
	hasDest := len(ic.WebhookURLs) > 0 || strings.TrimSpace(ic.SlackWebhookURL) != "" ||
		(strings.TrimSpace(ic.TelegramBotTokenEnv) != "" && strings.TrimSpace(ic.TelegramChatID) != "")
	if !hasDest {
		errs = append(errs, "integration.enabled requires at least one destination (webhook_urls, slack_webhook_url, or telegram_bot_token_env + telegram_chat_id)")
	}
	if strings.TrimSpace(ic.TelegramBotTokenEnv) != "" && strings.TrimSpace(ic.TelegramChatID) == "" {
		errs = append(errs, "integration.telegram_chat_id is required when telegram_bot_token_env is set")
	}
	if strings.TrimSpace(ic.TelegramChatID) != "" && strings.TrimSpace(ic.TelegramBotTokenEnv) == "" {
		errs = append(errs, "integration.telegram_bot_token_env is required when telegram_chat_id is set")
	}
	if strings.TrimSpace(ic.TelegramBotTokenEnv) != "" {
		if _, ok := os.LookupEnv(ic.TelegramBotTokenEnv); !ok {
			errs = append(errs, fmt.Sprintf("integration.telegram_bot_token_env %q is not set in the environment", ic.TelegramBotTokenEnv))
		}
	}
	for i, u := range ic.WebhookURLs {
		u = strings.TrimSpace(u)
		if u == "" {
			errs = append(errs, fmt.Sprintf("integration.webhook_urls[%d] is empty", i))
			continue
		}
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			errs = append(errs, fmt.Sprintf("integration.webhook_urls[%d] must be an http(s) URL", i))
		}
	}
	if s := strings.TrimSpace(ic.SlackWebhookURL); s != "" && !strings.HasPrefix(s, "https://") {
		errs = append(errs, "integration.slack_webhook_url must be an https URL")
	}
	return errs
}

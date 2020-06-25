package lambda

import (
	"fmt"
	"strings"

	"github.com/kafkaesque-io/pubsub-function/src/model"
)

const (
	// PulsarTrigger is pulsar input topic trigger
	PulsarTrigger = "pulsar-topic"

	// HTTPTrigger is http trigger
	HTTPTrigger = "http"

	// CronTrigger is time based cron trigger
	CronTrigger = "cron"
)

// ValidateFunctionConfig validates function config
func ValidateFunctionConfig(cfg *model.FunctionTopic) error {
	if !model.IsURL(cfg.PulsarURL) {
		return fmt.Errorf("not a URL %s", cfg.PulsarURL)
	}
	if strings.TrimSpace(cfg.Subscription) == "" {
		return fmt.Errorf("subscription name is missing")
	}
	if _, err := model.GetSubscriptionType(cfg.SubscriptionType); err != nil {
		return err
	}
	if _, err := model.GetInitialPosition(cfg.InitialPosition); err != nil {
		return err
	}
	return nil
}

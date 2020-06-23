package lambda

import (
	"fmt"
	"strings"

	"github.com/kafkaesque-io/pubsub-function/src/model"
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

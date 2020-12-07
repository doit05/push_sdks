package huawei

import (
	"errors"
	"regexp"

	model "push_sdks/common"
)

var (
	ttlPattern   = regexp.MustCompile("\\d+|\\d+[sS]|\\d+.\\d{1,9}|\\d+.\\d{1,9}[sS]")
	colorPattern = regexp.MustCompile("^#[0-9a-fA-F]{6}$")
)

func ValidateMessage(message *model.Message) error {
	if message == nil {
		return errors.New("message must not be null")
	}

	// validate field target, one of Token, Topic and Condition must be invoked
	if err := validateFieldTarget(message.Token, message.Topic, message.Condition); err != nil {
		return err
	}

	// validate android config
	if err := validateAndroidConfig(message.Android); err != nil {
		return err
	}

	// validate web common config
	if err := validateWebPushConfig(message.WebPush); err != nil {
		return err
	}
	return nil
}

func validateFieldTarget(token []string, strings ...string) error {
	count := 0
	if token != nil {
		count++
	}

	for _, s := range strings {
		if s != "" {
			count++
		}
	}

	if count == 1 {
		return nil
	}
	return errors.New("token, topics or condition must be choice one")
}

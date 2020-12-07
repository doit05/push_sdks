package huawei

import (
	"errors"

	model "push_sdks/common"
)

func validateWebPushConfig(webPushConfig *model.WebPushConfig) error {
	if webPushConfig == nil {
		return nil
	}

	if err := validateWebPushHeaders(webPushConfig.Headers); err != nil {
		return err
	}

	return validateWebPushNotification(webPushConfig.Notification)
}

func validateWebPushHeaders(headers *model.WebPushHeaders) error {
	if headers == nil {
		return nil
	}

	if headers.TTL != "" && !ttlPattern.MatchString(headers.TTL) {
		return errors.New("malformed ttl")
	}

	if headers.Urgency != "" &&
		headers.Urgency != UrgencyHigh &&
		headers.Urgency != UrgencyNormal &&
		headers.Urgency != UrgencyLow &&
		headers.Urgency != UrgencyVeryLow {
		return errors.New("priority must be 'high', 'normal', 'low' or 'very-low'")
	}
	return nil
}

func validateWebPushNotification(notification *model.WebPushNotification) error {
	if notification == nil {
		return nil
	}

	if err := validateWebPushAction(notification.Actions); err != nil {
		return err
	}

	if err := validateWebPushDirection(notification.Dir); err != nil {
		return err
	}
	return nil
}

func validateWebPushAction(actions []*model.WebPushAction) error {
	if actions == nil {
		return nil
	}

	for _, action := range actions {
		if action.Action == "" {
			return errors.New("web common action can't be empty")
		}
	}
	return nil
}

func validateWebPushDirection(dir string) error {
	if dir != DirAuto && dir != DirLtr && dir != DirRtl {
		return errors.New("web common dir must be 'auto', 'ltr', 'rtl'")
	}
	return nil
}

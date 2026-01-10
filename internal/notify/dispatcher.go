package notify

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Dispatcher sends notifications to all enabled webhooks.
type Dispatcher struct {
	webhookRepo *storage.WebhookRepo
	httpClient  *HTTPClient
	debug       bool
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher(webhookRepo *storage.WebhookRepo) *Dispatcher {
	return &Dispatcher{
		webhookRepo: webhookRepo,
		httpClient:  NewHTTPClient(),
	}
}

// SetDebug enables or disables debug output.
func (d *Dispatcher) SetDebug(debug bool) {
	d.debug = debug
}

// DispatchResult contains the result of dispatching to a single webhook.
type DispatchResult struct {
	WebhookName string
	Success     bool
	StatusCode  int
	Duration    time.Duration
	Error       error
}

// SendNotification sends a notification to all enabled webhooks.
func (d *Dispatcher) SendNotification(ctx context.Context, n *model.Notification) []DispatchResult {
	webhooks, err := d.webhookRepo.ListEnabled()
	if err != nil {
		return []DispatchResult{{
			WebhookName: "all",
			Success:     false,
			Error:       fmt.Errorf("failed to list webhooks: %w", err),
		}}
	}

	if len(webhooks) == 0 {
		return nil // No webhooks configured
	}

	// Send to all webhooks concurrently
	var wg sync.WaitGroup
	results := make([]DispatchResult, len(webhooks))

	for i, webhook := range webhooks {
		wg.Add(1)
		go func(idx int, wh *model.Webhook) {
			defer wg.Done()
			results[idx] = d.sendToWebhook(ctx, n, wh)
		}(i, webhook)
	}

	wg.Wait()
	return results
}

// sendToWebhook sends a notification to a single webhook.
func (d *Dispatcher) sendToWebhook(ctx context.Context, n *model.Notification, webhook *model.Webhook) DispatchResult {
	result := DispatchResult{
		WebhookName: webhook.Name,
	}

	// Get the appropriate formatter
	var formatter Formatter
	if webhook.Type == model.WebhookTypeGeneric && webhook.Template != "" {
		formatter = NewGenericFormatter(webhook.Template)
	} else {
		formatter = GetFormatter(webhook.Type)
	}

	// Format the notification
	payload, err := formatter.Format(n)
	if err != nil {
		result.Error = fmt.Errorf("failed to format notification: %w", err)
		d.updateWebhookStatus(webhook.Name, result.Error)
		return result
	}

	// Send the request
	sendResult := d.httpClient.Send(ctx, webhook.URL, formatter.ContentType(), payload)

	result.StatusCode = sendResult.StatusCode
	result.Duration = sendResult.Duration
	result.Error = sendResult.Error
	result.Success = sendResult.Error == nil

	// Update webhook last used status
	d.updateWebhookStatus(webhook.Name, sendResult.Error)

	return result
}

// updateWebhookStatus updates the last used timestamp and error for a webhook.
func (d *Dispatcher) updateWebhookStatus(name string, err error) {
	// Ignore errors from updating status - it's not critical
	_ = d.webhookRepo.UpdateLastUsed(name, err)
}

// SendToSingle sends a notification to a single webhook by name.
func (d *Dispatcher) SendToSingle(ctx context.Context, n *model.Notification, webhookName string) DispatchResult {
	webhook, err := d.webhookRepo.Get(webhookName)
	if err != nil {
		return DispatchResult{
			WebhookName: webhookName,
			Success:     false,
			Error:       fmt.Errorf("webhook not found: %w", err),
		}
	}

	return d.sendToWebhook(ctx, n, webhook)
}

// TestWebhook sends a test notification to a specific webhook.
func (d *Dispatcher) TestWebhook(ctx context.Context, webhookName string) DispatchResult {
	testNotification := model.NewNotification(
		model.NotifyTest,
		"Humantime Test",
		"This is a test notification from Humantime. If you see this, your webhook is configured correctly!",
	).WithField("Webhook", webhookName).WithField("Time", time.Now().Format("3:04 PM"))

	return d.SendToSingle(ctx, testNotification, webhookName)
}

// HasEnabledWebhooks returns true if there are any enabled webhooks.
func (d *Dispatcher) HasEnabledWebhooks() bool {
	webhooks, err := d.webhookRepo.ListEnabled()
	if err != nil {
		return false
	}
	return len(webhooks) > 0
}

// CountEnabledWebhooks returns the number of enabled webhooks.
func (d *Dispatcher) CountEnabledWebhooks() int {
	webhooks, err := d.webhookRepo.ListEnabled()
	if err != nil {
		return 0
	}
	return len(webhooks)
}

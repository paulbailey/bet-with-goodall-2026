package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	webpush "github.com/SherClockHolmes/webpush-go"
)

// Notification is the payload the service worker receives and renders. Kept in
// sync with the PushPayload interface in web/src/sw.ts.
type Notification struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
}

// storedSub mirrors one row of the push-subscriptions DynamoDB table written by
// the subscription Lambda.
type storedSub struct {
	ID       string
	Endpoint string
	P256dh   string
	Auth     string
}

// pushNotifier sends Web Push notifications to every stored subscription and
// prunes any the push service reports as gone.
type pushNotifier struct {
	ddb     *dynamodb.Client
	table   string
	public  string
	private string
	subject string
	logger  *slog.Logger
}

// setupPush builds the notifier when a VAPID private key and a subscription
// table are configured (and we're in S3 mode, where a region and AWS creds are
// available). Returns nil to disable pushes; any setup issue degrades gracefully.
func setupPush(ctx context.Context, env Env, logger *slog.Logger) *pushNotifier {
	if env.VapidPrivate == "" || env.PushTable == "" {
		logger.Info("push notifications disabled (no VAPID_PRIVATE_KEY or PUSH_TABLE)")
		return nil
	}
	if env.LocalOutput != "" {
		logger.Info("push notifications disabled (local mode)")
		return nil
	}
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(env.Region))
	if err != nil {
		logger.Warn("push notifications disabled (AWS config)", "err", err)
		return nil
	}
	logger.Info("push notifications enabled", "table", env.PushTable)
	return &pushNotifier{
		ddb:     dynamodb.NewFromConfig(cfg),
		table:   env.PushTable,
		public:  env.VapidPublic,
		private: env.VapidPrivate,
		subject: env.VapidSubject,
		logger:  logger,
	}
}

// send delivers one notification to every current subscription. Subscriptions
// the push service reports as gone (404/410) are deleted so the table doesn't
// accumulate dead endpoints.
func (p *pushNotifier) send(ctx context.Context, n Notification) {
	subs, err := p.loadSubscriptions(ctx)
	if err != nil {
		p.logger.Warn("push: load subscriptions failed", "err", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	payload, err := json.Marshal(n)
	if err != nil {
		p.logger.Error("push: marshal payload failed", "err", err)
		return
	}

	sent, pruned := 0, 0
	for _, s := range subs {
		gone, err := p.deliver(ctx, s, payload)
		switch {
		case gone:
			if err := p.prune(ctx, s.ID); err != nil {
				p.logger.Warn("push: prune failed", "err", err)
			} else {
				pruned++
			}
		case err != nil:
			p.logger.Warn("push: deliver failed", "err", err)
		default:
			sent++
		}
	}
	p.logger.Info("push sent", "tag", n.Tag, "delivered", sent, "pruned", pruned, "total", len(subs))
}

// deliver posts one push. It returns gone=true when the subscription is no
// longer valid (the caller should prune it).
func (p *pushNotifier) deliver(ctx context.Context, s storedSub, payload []byte) (gone bool, err error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: s.Endpoint,
		Keys:     webpush.Keys{P256dh: s.P256dh, Auth: s.Auth},
	}, &webpush.Options{
		Subscriber:      p.subject,
		VAPIDPublicKey:  p.public,
		VAPIDPrivateKey: p.private,
		TTL:             86400, // a day — a match result is still worth seeing late
		Urgency:         webpush.UrgencyHigh,
	})
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return true, nil
	case resp.StatusCode >= 300:
		return false, fmt.Errorf("push service returned HTTP %d", resp.StatusCode)
	}
	return false, nil
}

// loadSubscriptions scans the table, following pagination so a larger group
// isn't silently truncated.
func (p *pushNotifier) loadSubscriptions(ctx context.Context) ([]storedSub, error) {
	var out []storedSub
	var start map[string]ddbtypes.AttributeValue
	for {
		page, err := p.ddb.Scan(ctx, &dynamodb.ScanInput{
			TableName:         aws.String(p.table),
			ExclusiveStartKey: start,
		})
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", p.table, err)
		}
		for _, item := range page.Items {
			s := storedSub{
				ID:       stringAttr(item, "id"),
				Endpoint: stringAttr(item, "endpoint"),
				P256dh:   stringAttr(item, "p256dh"),
				Auth:     stringAttr(item, "auth"),
			}
			if s.Endpoint == "" || s.P256dh == "" || s.Auth == "" {
				continue // malformed row, skip
			}
			out = append(out, s)
		}
		if len(page.LastEvaluatedKey) == 0 {
			break
		}
		start = page.LastEvaluatedKey
	}
	return out, nil
}

func (p *pushNotifier) prune(ctx context.Context, id string) error {
	_, err := p.ddb.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(p.table),
		Key:       map[string]ddbtypes.AttributeValue{"id": &ddbtypes.AttributeValueMemberS{Value: id}},
	})
	return err
}

// stringAttr reads a DynamoDB string attribute, returning "" when absent or of
// another type.
func stringAttr(item map[string]ddbtypes.AttributeValue, key string) string {
	if sv, ok := item[key].(*ddbtypes.AttributeValueMemberS); ok {
		return sv.Value
	}
	return ""
}

// Package channel defines how a new Conversation message gets delivered
// somewhere outside localchat — the same pluggable "delivery channel"
// concept used by every real conversations/messaging product (Twilio
// Conversations, Intercom, Front): a conversation can be bound to zero or
// more channels, and every channel fires on every new message. Today the
// only implementation is Webhook; SMS/email channels can be added later by
// satisfying the same Channel interface — nothing else in this package, or
// in the HTTP layer, needs to change.
package channel

import (
    "context"
    "log"

    "localchat/internal/store"
)

// Channel delivers a single message to wherever a bound Channel record
// points. Implementations must be safe for concurrent use and should treat
// delivery failures as non-fatal to the caller (log and return an error;
// never panic) — a slow or broken external endpoint must never block the
// conversation itself.
type Channel interface {
    // Type returns the channel type string this implementation handles
    // (matches store.Channel.Type), used by the dispatcher to route.
    Type() string
    Deliver(ctx context.Context, conv *store.Conversation, msg *store.Message, config map[string]interface{}) error
}

// Dispatcher fires every channel bound to a conversation whenever a new
// message is posted to it.
type Dispatcher struct {
    store      *store.Store
    registered map[string]Channel
}

func NewDispatcher(s *store.Store, channels ...Channel) *Dispatcher {
    d := &Dispatcher{store: s, registered: make(map[string]Channel, len(channels))}
    for _, c := range channels {
        d.registered[c.Type()] = c
    }
    return d
}

// Dispatch looks up every channel bound to conv (specific + global) and
// delivers msg to each concurrently. Delivery is fire-and-forget from the
// caller's perspective — errors are logged, not returned, so a slow or
// down webhook receiver never delays the API response to whoever posted
// the message. Callers that need delivery confirmation should have their
// channel's receiving side report status back through its own means (e.g.
// a webhook receiver that itself calls back into the Conversations API).
func (d *Dispatcher) Dispatch(ctx context.Context, conv *store.Conversation, msg *store.Message) {
    channels, err := d.store.ChannelsFor(ctx, conv.ID)
    if err != nil {
        log.Printf("channel: lookup failed for conversation %s: %v", conv.ID, err)
        return
    }
    for _, ch := range channels {
        impl, ok := d.registered[ch.Type]
        if !ok {
            log.Printf("channel: no implementation registered for type %q (channel %s) — skipping", ch.Type, ch.ID)
            continue
        }
        go func(impl Channel, config map[string]interface{}, channelID string) {
            if err := impl.Deliver(context.Background(), conv, msg, config); err != nil {
                log.Printf("channel: delivery failed (channel %s, type %s): %v", channelID, impl.Type(), err)
            }
        }(impl, ch.Config, ch.ID)
    }
}

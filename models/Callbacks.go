package models

//SubscriberNotifyCallback callback for user notifications
type SubscriberNotifyCallback interface {
	OnWebhookReceive(*Webhook, *Source)
}

//NotifyCallback callback for Notify
type NotifyCallback interface {
	OnSuccess(Subscription)
	OnError(Subscription, Source, Webhook)
	OnUnsubscribe(Subscription)
}

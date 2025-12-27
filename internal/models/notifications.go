package models

type PushSubscription struct {
	Id           int    `json:"id"`
	Endpoint     string `json:"endpoint"`
	P256dh       string `json:"p256dh"`
	Auth         string `json:"auth"`
	ReminderDay  int    `json:"reminder_day"`  // 0-6 (Sunday-Saturday)
	ReminderTime string `json:"reminder_time"` // "HH:MM"
	Timezone     string `json:"timezone"`      // e.g. "Europe/Stockholm"
}

type PushSubscriptionRequest struct {
	Subscription PushSubscription `json:"subscription"`
	ReminderDay  int              `json:"reminder_day"`
	ReminderTime string           `json:"reminder_time"`
	Timezone     string           `json:"timezone"`
}

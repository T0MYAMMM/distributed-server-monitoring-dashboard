package alerts

import "github.com/thomasstefen/server-monitor/internal/domain"

// MultiNotifier fans an alert out to several notifiers (e.g. the legacy env
// webhook plus every enabled notification channel). Each delegate is expected
// to be non-blocking; nil delegates are skipped.
type MultiNotifier struct {
	targets []Notifier
}

// NewMultiNotifier combines notifiers, dropping nil entries.
func NewMultiNotifier(targets ...Notifier) Notifier {
	kept := make([]Notifier, 0, len(targets))
	for _, t := range targets {
		if t != nil {
			kept = append(kept, t)
		}
	}
	return MultiNotifier{targets: kept}
}

// Notify forwards the alert to every delegate.
func (m MultiNotifier) Notify(a domain.Alert) {
	for _, t := range m.targets {
		t.Notify(a)
	}
}

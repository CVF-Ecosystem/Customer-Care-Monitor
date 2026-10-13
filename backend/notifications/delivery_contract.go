package notifications

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

const (
	deliveryPending      = "pending"
	deliverySending      = "sending"
	deliverySent         = "sent"
	deliveryFailedStatus = "failed"
	deliveryUnknown      = "unknown"
)

type deliveryError struct {
	status string
	stage  string
}

func (e *deliveryError) Error() string { return "delivery " + e.status + " at " + e.stage }
func failedDelivery(stage string) error {
	return &deliveryError{status: deliveryFailedStatus, stage: stage}
}
func unknownDelivery(stage string) error {
	return &deliveryError{status: deliveryUnknown, stage: stage}
}

func deliveryOutcome(err error) (string, string) {
	if err == nil {
		return deliverySent, ""
	}
	if typed, ok := err.(*deliveryError); ok {
		if typed.status == deliveryFailedStatus || typed.status == deliveryUnknown {
			return typed.status, typed.Error()
		}
	}
	return deliveryUnknown, "delivery outcome unknown"
}

type outputIdentity struct {
	Type       string   `json:"type"`
	Endpoint   string   `json:"endpoint"`
	Target     string   `json:"target"`
	Recipients []string `json:"recipients,omitempty"`
	Template   string   `json:"template"`
	Ordinal    int      `json:"ordinal"`
}

type resolvedOutput struct {
	Config   OutputConfig
	Key      string
	Identity outputIdentity
}

func resolveOutputs(configs []OutputConfig) ([]resolvedOutput, string, error) {
	type indexed struct {
		config    OutputConfig
		identity  outputIdentity
		canonical string
	}
	items := make([]indexed, 0, len(configs))
	for _, cfg := range configs {
		typ := strings.ToLower(strings.TrimSpace(cfg.Type))
		id := outputIdentity{Type: typ, Template: "default"}
		if cfg.Template == "custom" && cfg.CustomTemplate != "" {
			id.Template = "custom:" + digest([]byte(cfg.CustomTemplate))
		}
		switch typ {
		case "telegram":
			id.Endpoint = "telegram"
			id.Target = strings.TrimSpace(cfg.ChatID)
		case "email":
			id.Endpoint = strings.ToLower(strings.TrimSpace(cfg.SMTPHost)) + ":" + strings.TrimSpace(jsonNumber(cfg.SMTPPort))
			id.Target = strings.ToLower(strings.TrimSpace(cfg.From))
			for _, recipient := range splitComma(cfg.To) {
				value := strings.ToLower(strings.TrimSpace(recipient))
				if value != "" {
					id.Recipients = append(id.Recipients, value)
				}
			}
			sort.Strings(id.Recipients)
		default:
			id.Endpoint = "unsupported"
			id.Target = typ
		}
		base, err := json.Marshal(id)
		if err != nil {
			return nil, "", err
		}
		items = append(items, indexed{config: cfg, identity: id, canonical: string(base)})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].canonical < items[j].canonical })
	counts := make(map[string]int)
	resolved := make([]resolvedOutput, 0, len(items))
	keys := make([]string, 0, len(items))
	for _, item := range items {
		ordinal := counts[item.canonical]
		counts[item.canonical] = ordinal + 1
		item.identity.Ordinal = ordinal
		encoded, err := json.Marshal(item.identity)
		if err != nil {
			return nil, "", err
		}
		key := digest(encoded)
		resolved = append(resolved, resolvedOutput{Config: item.config, Key: key, Identity: item.identity})
		keys = append(keys, key)
	}
	configJSON, err := json.Marshal(keys)
	if err != nil {
		return nil, "", err
	}
	return resolved, digest(configJSON), nil
}

func jsonNumber(value int) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

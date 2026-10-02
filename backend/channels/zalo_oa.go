package channels

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	zaloAPIBaseV2 = "https://openapi.zalo.me/v2.0/oa"
	zaloAPIBaseV3 = "https://openapi.zalo.me/v3.0/oa"
	zaloOAuthURL  = "https://oauth.zaloapp.com/v4/oa/access_token"

	// zaloConversationPageSize is the listrecentchat count (Zalo's documented maximum is 10).
	zaloConversationPageSize = 10
	// zaloMaxConversationPages bounds one enumeration, counting the terminal empty page.
	// Reaching it is incomplete coverage, never a truncated success (CCMAI-RUNTIME-024).
	zaloMaxConversationPages = 500
	// zaloMaxResponseBytes bounds every response body read before decoding.
	zaloMaxResponseBytes = 8 << 20
	// zaloMaxUnixMilli is 9999-12-31T23:59:59.999Z, the last time the calendar accepts.
	zaloMaxUnixMilli = 253402300799999
)

// zaloMessagePageSize is the conversation count (Zalo's documented maximum is 10).
const zaloMessagePageSize = 10

// zaloMaxMessagePages bounds one message traversal, counting the terminal empty page. Reaching it
// is incomplete coverage, never a truncated success (CCMAI-RUNTIME-032).
const zaloMaxMessagePages = 500

// ErrZaloMessageCoverageIncomplete marks a message traversal that did not reach an explicit empty
// page or could not trust a page. Rows returned with it are diagnostic only.
var ErrZaloMessageCoverageIncomplete = errors.New("zalo message coverage incomplete")

// ErrZaloCoverageIncomplete marks a conversation enumeration that did not reach an explicit
// empty page or could not trust a page. Rows returned with it are diagnostic only.
var ErrZaloCoverageIncomplete = errors.New("zalo conversation coverage incomplete")

// zaloSafeError shows only a fixed description while keeping its cause reachable through
// errors.Is/As. Transport, callback and decode causes can carry URLs, tokens or provider text.
type zaloSafeError struct {
	msg   string
	cause error
}

func (e *zaloSafeError) Error() string { return e.msg }
func (e *zaloSafeError) Unwrap() error { return e.cause }

func zaloSafe(msg string, cause error) error { return &zaloSafeError{msg: msg, cause: cause} }

// zaloRequestFailure classifies a transport error without exposing its text.
func zaloRequestFailure(ctx context.Context, what string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return zaloSafe(what+" cancelled", ctxErr)
	}
	return zaloSafe(what+" failed", err)
}

// readZaloBody reads at most zaloMaxResponseBytes and always closes the body.
func readZaloBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, zaloMaxResponseBytes+1))
	if err != nil {
		return nil, zaloSafe("zalo response read failed", err)
	}
	if len(body) > zaloMaxResponseBytes {
		return nil, errors.New("zalo response exceeds the 8 MiB limit")
	}
	return body, nil
}

// zaloEnvelopeError reads the numeric "error" field of a JSON object body. present is false when
// the field is absent; a non-object body or a non-integral code is a malformed envelope.
func zaloEnvelopeError(body []byte) (code int64, present bool, err error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return 0, false, errors.New("zalo response is not a JSON object")
	}
	raw, ok := fields["error"]
	if !ok {
		return 0, false, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var value interface{}
	if err := dec.Decode(&value); err != nil {
		return 0, true, errors.New("zalo response has a malformed error code")
	}
	// Decoding directly into json.Number also accepts a quoted numeric string. The
	// envelope requires an actual JSON number before it can authorize token refresh.
	n, ok := value.(json.Number)
	if !ok {
		return 0, true, errors.New("zalo response has a malformed error code")
	}
	code, convErr := n.Int64()
	if convErr != nil {
		return 0, true, errors.New("zalo response has a malformed error code")
	}
	return code, true, nil
}

// ZaloOACredentials holds the credentials needed for Zalo OA API.
type ZaloOACredentials struct {
	AppID        string `json:"app_id"`
	AppSecret    string `json:"app_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	OAId         string `json:"oa_id"`
}

// OnTokenRefresh is called when tokens are refreshed — caller should persist new creds.
type OnTokenRefresh func(newAccessToken, newRefreshToken string) error

type ZaloOAAdapter struct {
	creds          ZaloOACredentials
	client         *http.Client
	mu             sync.Mutex
	onTokenRefresh OnTokenRefresh
}

func NewZaloOAAdapter(creds ZaloOACredentials) *ZaloOAAdapter {
	return &ZaloOAAdapter{
		creds:  creds,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (z *ZaloOAAdapter) SetTokenRefreshCallback(cb OnTokenRefresh) {
	z.onTokenRefresh = cb
}

// refreshToken performs Zalo token refresh (single-use rotation).
func (z *ZaloOAAdapter) refreshToken(ctx context.Context) error {
	z.mu.Lock()
	defer z.mu.Unlock()

	form := url.Values{
		"refresh_token": {z.creds.RefreshToken},
		"app_id":        {z.creds.AppID},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", zaloOAuthURL, nil)
	if err != nil {
		return zaloSafe("create zalo token refresh request failed", err)
	}
	req.URL.RawQuery = form.Encode()
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", z.creds.AppSecret)

	resp, err := z.client.Do(req)
	if err != nil {
		return zaloRequestFailure(ctx, "zalo token refresh request", err)
	}
	body, err := readZaloBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("zalo token refresh error: http %d", resp.StatusCode)
	}
	code, _, err := zaloEnvelopeError(body)
	if err != nil {
		return err
	}
	if code != 0 {
		// The provider message is not shown: it can echo request values.
		return fmt.Errorf("zalo token refresh error %d", code)
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return errors.New("zalo token refresh decode failed")
	}
	// A rotated pair must be complete before it is persisted or used: the old refresh token
	// is single-use and may already be spent.
	if strings.TrimSpace(result.AccessToken) == "" || strings.TrimSpace(result.RefreshToken) == "" {
		return errors.New("zalo token refresh returned an incomplete token pair")
	}
	if err := ctx.Err(); err != nil {
		return zaloSafe("zalo token refresh cancelled", err)
	}

	if z.onTokenRefresh != nil {
		if err := z.onTokenRefresh(result.AccessToken, result.RefreshToken); err != nil {
			return zaloSafe("zalo token refresh persistence failed", err)
		}
	}
	z.creds.AccessToken = result.AccessToken
	z.creds.RefreshToken = result.RefreshToken

	return nil
}

// doRequestRaw makes an authenticated Zalo API request and returns the bounded body of a
// successful (2xx, error 0 or absent) response. On error=-216 it refreshes the token at most
// once and retries the same request once. Errors never contain URLs, bodies, provider messages
// or tokens.
func (z *ZaloOAAdapter) doRequestRaw(ctx context.Context, method, apiURL string, params map[string]interface{}) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, apiURL, nil)
		if err != nil {
			return nil, zaloSafe("create zalo api request failed", err)
		}

		// Zalo API: params go as JSON-encoded `data` query param
		if params != nil {
			q := req.URL.Query()
			paramsJSON, _ := json.Marshal(params)
			q.Set("data", string(paramsJSON))
			req.URL.RawQuery = q.Encode()
		}

		z.mu.Lock()
		token := z.creds.AccessToken
		z.mu.Unlock()
		req.Header.Set("access_token", token)

		resp, err := z.client.Do(req)
		if err != nil {
			return nil, zaloRequestFailure(ctx, "zalo api request", err)
		}
		body, err := readZaloBody(resp)
		if err != nil {
			return nil, err
		}

		log.Printf("[zalo] API %s: status=%d len=%d", path.Base(req.URL.Path), resp.StatusCode, len(body))

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, fmt.Errorf("zalo api error: http %d", resp.StatusCode)
		}
		code, _, err := zaloEnvelopeError(body)
		if err != nil {
			return nil, err
		}

		// Check for token expired error (error=-216)
		if code == -216 && attempt == 0 {
			if refreshErr := z.refreshToken(ctx); refreshErr != nil {
				return nil, fmt.Errorf("token refresh failed: %w", refreshErr)
			}
			continue
		}
		if code != 0 {
			return nil, fmt.Errorf("zalo api error %d", code)
		}
		return body, nil
	}
	return nil, fmt.Errorf("zalo api failed after retry")
}

// doRequest is doRequestRaw decoded into a generic map (numbers as float64), the form
// FetchMessages and HealthCheck have always consumed.
func (z *ZaloOAAdapter) doRequest(ctx context.Context, method, apiURL string, params map[string]interface{}) (map[string]interface{}, error) {
	body, err := z.doRequestRaw(ctx, method, apiURL, params)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.New("zalo api decode failed")
	}
	return result, nil
}

// zaloConversationRow is one validated listrecentchat row.
type zaloConversationRow struct {
	userID string
	name   string
	millis int64
	raw    map[string]interface{}
}

// zaloInt64 accepts only an integral JSON number (decoded with UseNumber).
func zaloInt64(v interface{}) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := n.Int64()
	return i, err == nil
}

// parseZaloConversationPage validates one listrecentchat page: a JSON object with error 0, an
// explicit array as `data` or `data.data`, and every physical row an object with src 0 or 1,
// the selected customer's nonempty string ID and a positive millisecond time. Numbers are kept
// exact (UseNumber); IDs are never converted through float64.
func parseZaloConversationPage(body []byte) ([]zaloConversationRow, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var env map[string]interface{}
	if err := dec.Decode(&env); err != nil || env == nil {
		return nil, fmt.Errorf("%w: page is not a JSON object", ErrZaloCoverageIncomplete)
	}
	if code, ok := zaloInt64(env["error"]); !ok || code != 0 {
		return nil, fmt.Errorf("%w: page has no successful error code", ErrZaloCoverageIncomplete)
	}
	var items []interface{}
	switch data := env["data"].(type) {
	case []interface{}:
		items = data
	case map[string]interface{}:
		arr, ok := data["data"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: page has no data array", ErrZaloCoverageIncomplete)
		}
		items = arr
	default:
		return nil, fmt.Errorf("%w: page has no data array", ErrZaloCoverageIncomplete)
	}

	rows := make([]zaloConversationRow, 0, len(items))
	for _, item := range items {
		conv, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%w: page has a malformed row", ErrZaloCoverageIncomplete)
		}
		// src=0: the OA sent the last message, the customer is "to"; src=1: the customer sent it.
		src, ok := zaloInt64(conv["src"])
		if !ok || (src != 0 && src != 1) {
			return nil, fmt.Errorf("%w: row has an invalid src", ErrZaloCoverageIncomplete)
		}
		idKey, nameKey := "from_id", "from_display_name"
		if src == 0 {
			idKey, nameKey = "to_id", "to_display_name"
		}
		userID, _ := conv[idKey].(string)
		if userID == "" {
			return nil, fmt.Errorf("%w: row has no customer id", ErrZaloCoverageIncomplete)
		}
		millis, ok := zaloInt64(conv["time"])
		if !ok || millis <= 0 || millis > zaloMaxUnixMilli {
			return nil, fmt.Errorf("%w: row has an invalid time", ErrZaloCoverageIncomplete)
		}
		name, _ := conv[nameKey].(string)
		rows = append(rows, zaloConversationRow{userID: userID, name: name, millis: millis, raw: conv})
	}
	return rows, nil
}

// FetchRecentConversations enumerates every customer conversation whose last message is at or
// after `since` (CCMAI-RUNTIME-024). Pages are requested by absolute row offset; every
// nonempty page, short or not, advances the offset by its physical row count, and only an
// explicit empty array ends the enumeration successfully. Every physical row is validated
// before filtering; no ordering is assumed. Duplicate customers keep the newest time and its
// row (the first seen wins a tie). A repeated page, cancellation, the page budget, any request
// or page error, or a positive limit the window exceeds returns ErrZaloCoverageIncomplete with
// the rows seen so far, which are diagnostic only.
func (z *ZaloOAAdapter) FetchRecentConversations(ctx context.Context, since time.Time, limit int) ([]SyncedConversation, error) {
	byID := make(map[string]*SyncedConversation)
	var order []string
	collected := func() []SyncedConversation {
		out := make([]SyncedConversation, 0, len(order))
		for _, id := range order {
			out = append(out, *byID[id])
		}
		return out
	}
	seenPages := make(map[string]bool)
	offset := 0

	for page := 0; ; page++ {
		if page >= zaloMaxConversationPages {
			return collected(), fmt.Errorf("%w: page budget of %d reached before an empty page", ErrZaloCoverageIncomplete, zaloMaxConversationPages)
		}
		if err := ctx.Err(); err != nil {
			return collected(), fmt.Errorf("%w: %w", ErrZaloCoverageIncomplete, zaloSafe("enumeration cancelled", err))
		}

		body, err := z.doRequestRaw(ctx, "GET", zaloAPIBaseV2+"/listrecentchat", map[string]interface{}{
			"offset": offset,
			"count":  zaloConversationPageSize,
		})
		if err != nil {
			return collected(), fmt.Errorf("%w: page %d: %w", ErrZaloCoverageIncomplete, page+1, err)
		}
		rows, err := parseZaloConversationPage(body)
		if err != nil {
			return collected(), fmt.Errorf("%w (page %d)", err, page+1)
		}
		if err := ctx.Err(); err != nil {
			return collected(), fmt.Errorf("%w: %w", ErrZaloCoverageIncomplete, zaloSafe("enumeration cancelled", err))
		}
		if len(rows) == 0 {
			return collected(), nil
		}

		// A page whose rows (customer and time) equal an earlier page means the offset is not
		// moving through new data.
		var fp strings.Builder
		for _, r := range rows {
			fmt.Fprintf(&fp, "%s@%d\n", r.userID, r.millis)
		}
		if seenPages[fp.String()] {
			return collected(), fmt.Errorf("%w: page %d repeats an earlier page", ErrZaloCoverageIncomplete, page+1)
		}
		seenPages[fp.String()] = true

		for _, r := range rows {
			lastMsgAt := time.UnixMilli(r.millis)
			if !since.IsZero() && lastMsgAt.Before(since) {
				continue
			}
			if existing, ok := byID[r.userID]; ok {
				if lastMsgAt.After(existing.LastMessageAt) {
					existing.CustomerName = r.name
					existing.LastMessageAt = lastMsgAt
					existing.Metadata = r.raw
				}
				continue
			}
			byID[r.userID] = &SyncedConversation{
				ExternalID:     r.userID, // Zalo uses user_id as conversation key
				ExternalUserID: r.userID,
				CustomerName:   r.name,
				LastMessageAt:  lastMsgAt,
				Metadata:       r.raw,
			}
			order = append(order, r.userID)
		}
		if limit > 0 && len(order) > limit {
			return collected()[:limit], fmt.Errorf("%w: more than %d eligible conversations in the window", ErrZaloCoverageIncomplete, limit)
		}
		offset += len(rows)
	}
}

// zaloMessageRow is one validated conversation row with its mapped message. key is the canonical
// encoding of the raw row, used to tell an exact repeat from a conflicting row with the same ID.
type zaloMessageRow struct {
	id  string
	key string
	msg SyncedMessage
}

// zaloOptionalString reads a consumed string field that may be absent. A present value of any
// other type, including null, makes the row invalid.
func zaloOptionalString(m map[string]interface{}, key string) (value string, ok bool) {
	v, present := m[key]
	if !present {
		return "", true
	}
	s, isString := v.(string)
	return s, isString
}

func zaloMessageInvalid(what string) error {
	return fmt.Errorf("%w: %s", ErrZaloMessageCoverageIncomplete, what)
}

// parseZaloMessagePage validates one conversation page: a JSON object with a numeric error 0, an
// explicit array as data or data.data, and every physical row an object with a nonblank string
// message_id, src 0 or 1 and a positive integral millisecond time. Numbers stay exact (UseNumber).
// Every row is validated and mapped before any deduplication.
func parseZaloMessagePage(body []byte) ([]zaloMessageRow, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var env map[string]interface{}
	if err := dec.Decode(&env); err != nil || env == nil {
		return nil, zaloMessageInvalid("page is not a JSON object")
	}
	if code, ok := zaloInt64(env["error"]); !ok || code != 0 {
		return nil, zaloMessageInvalid("page has no successful error code")
	}
	var items []interface{}
	switch data := env["data"].(type) {
	case []interface{}:
		items = data
	case map[string]interface{}:
		arr, ok := data["data"].([]interface{})
		if !ok {
			return nil, zaloMessageInvalid("page has no data array")
		}
		items = arr
	default:
		return nil, zaloMessageInvalid("page has no data array")
	}

	rows := make([]zaloMessageRow, 0, len(items))
	for _, item := range items {
		raw, ok := item.(map[string]interface{})
		if !ok {
			return nil, zaloMessageInvalid("page has a malformed row")
		}
		id, ok := raw["message_id"].(string)
		if !ok || strings.TrimSpace(id) == "" {
			return nil, zaloMessageInvalid("row has no message id")
		}
		src, ok := zaloInt64(raw["src"])
		if !ok || (src != 0 && src != 1) {
			return nil, zaloMessageInvalid("row has an invalid src")
		}
		millis, ok := zaloInt64(raw["time"])
		if !ok || millis <= 0 || millis > zaloMaxUnixMilli {
			return nil, zaloMessageInvalid("row has an invalid time")
		}
		content, ok1 := zaloOptionalString(raw, "message")
		msgType, ok2 := zaloOptionalString(raw, "type")
		from, ok3 := zaloOptionalString(raw, "from_display_name")
		topURL, ok4 := zaloOptionalString(raw, "url")
		thumb, ok5 := zaloOptionalString(raw, "thumb")
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
			return nil, zaloMessageInvalid("row has a mistyped field")
		}
		var firstLink map[string]interface{}
		if v, present := raw["links"]; present {
			links, ok := v.([]interface{})
			if !ok {
				return nil, zaloMessageInvalid("row has a malformed links field")
			}
			for i, l := range links {
				obj, ok := l.(map[string]interface{})
				if !ok {
					return nil, zaloMessageInvalid("row has a malformed link")
				}
				if i == 0 {
					firstLink = obj
				}
			}
		}
		// The first object link overrides the top-level URL and names the attachment, as the
		// inherited mapping did; a present string value counts even when it is empty.
		var linkURL, linkName string
		var linkURLSet, linkNameSet bool
		if firstLink != nil {
			okU, okN := true, true
			if v, present := firstLink["url"]; present {
				linkURL, okU = v.(string)
				linkURLSet = okU
			}
			if v, present := firstLink["name"]; present {
				linkName, okN = v.(string)
				linkNameSet = okN
			}
			if !okU || !okN {
				return nil, zaloMessageInvalid("row has a mistyped link field")
			}
		}

		senderType, senderName := "customer", from
		if src == 0 {
			senderType, senderName = "agent", "OA"
		}
		msg := SyncedMessage{
			ExternalID:  id,
			SenderType:  senderType,
			SenderName:  senderName,
			Content:     content,
			ContentType: "text",
			SentAt:      time.UnixMilli(millis),
			RawData:     raw,
		}
		if msgType != "" && msgType != "text" {
			msg.ContentType = msgType
			aURL, aName := "", ""
			if topURL != "" {
				aURL = topURL
			} else if thumb != "" {
				aURL = thumb
			}
			if linkURLSet {
				aURL = linkURL
			}
			if linkNameSet {
				aName = linkName
			}
			if aURL != "" {
				if aName == "" {
					aName = fmt.Sprintf("%s-%s", msgType, id)
				}
				msg.Attachments = append(msg.Attachments, Attachment{Type: msgType, URL: aURL, Name: aName})
			}
		}
		key, err := json.Marshal(raw)
		if err != nil {
			return nil, zaloMessageInvalid("row cannot be encoded")
		}
		rows = append(rows, zaloMessageRow{id: id, key: string(key), msg: msg})
	}
	if dec.More() {
		return nil, zaloMessageInvalid("page has trailing data")
	}
	return rows, nil
}

// doMessageRequest is the message-only authenticated GET (CCMAI-RUNTIME-032). It blocks every
// redirect before a destination is requested, keeps the injected transport and timeout by using a
// copy of the client, bounds the body, and shows no URL, query, body, provider text or token. As in
// doRequestRaw, error=-216 refreshes the token at most once and retries the same offset once; the
// shared refresh path, credentials and OAuth protocol are used unchanged.
func (z *ZaloOAAdapter) doMessageRequest(ctx context.Context, conversationID string, offset int) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, zaloAPIBaseV2+"/conversation", nil)
		if err != nil {
			return nil, zaloSafe("create zalo api request failed", err)
		}
		params, _ := json.Marshal(map[string]interface{}{
			"user_id": conversationID,
			"offset":  offset,
			"count":   zaloMessagePageSize,
		})
		q := req.URL.Query()
		q.Set("data", string(params))
		req.URL.RawQuery = q.Encode()

		z.mu.Lock()
		token := z.creds.AccessToken
		z.mu.Unlock()
		req.Header.Set("access_token", token)

		client := *z.client
		client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		resp, err := client.Do(req)
		if err != nil {
			return nil, zaloRequestFailure(ctx, "zalo api request", err)
		}
		body, err := readZaloBody(resp)
		if err != nil {
			return nil, err
		}

		log.Printf("[zalo] API %s: status=%d len=%d", path.Base(req.URL.Path), resp.StatusCode, len(body))

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return nil, fmt.Errorf("zalo api error: http %d", resp.StatusCode)
		}
		code, _, err := zaloEnvelopeError(body)
		if err != nil {
			return nil, err
		}
		if code == -216 && attempt == 0 {
			if refreshErr := z.refreshToken(ctx); refreshErr != nil {
				return nil, fmt.Errorf("token refresh failed: %w", refreshErr)
			}
			continue
		}
		if code != 0 {
			return nil, fmt.Errorf("zalo api error %d", code)
		}
		return body, nil
	}
	return nil, fmt.Errorf("zalo api failed after retry")
}

// FetchMessages reads a customer's whole message history (CCMAI-RUNTIME-032). Pages of
// zaloMessagePageSize are requested by absolute row offset; every nonempty page, short or not,
// advances the offset by its physical row count, and only a successful explicit empty array ends
// the traversal. Every physical row is validated before deduplication. The result has one message
// per message_id, sorted ascending by time with ties in first-seen order; an identical repeated
// row collapses, a conflicting row with the same ID fails. since is deliberately not applied: the
// analyzer needs full history and the store deduplicates. A repeated page, cancellation, the page
// budget, a request or page error or a conflicting row returns ErrZaloMessageCoverageIncomplete
// with the rows seen so far, which are diagnostic only. A nil error means this local contract
// completed; it does not establish provider ordering, retention, snapshot stability or live
// compatibility.
func (z *ZaloOAAdapter) FetchMessages(ctx context.Context, conversationID string, since time.Time) ([]SyncedMessage, error) {
	var order []string
	byID := make(map[string]zaloMessageRow)
	collected := func() []SyncedMessage {
		out := make([]SyncedMessage, 0, len(order))
		for _, id := range order {
			out = append(out, byID[id].msg)
		}
		sort.SliceStable(out, func(i, j int) bool { return out[i].SentAt.Before(out[j].SentAt) })
		return out
	}
	if strings.TrimSpace(conversationID) == "" {
		return nil, zaloMessageInvalid("conversation id is empty")
	}
	seenPages := make(map[[sha256.Size]byte]bool)
	offset := 0

	for page := 0; ; page++ {
		if page >= zaloMaxMessagePages {
			return collected(), fmt.Errorf("%w: page budget of %d reached before an empty page", ErrZaloMessageCoverageIncomplete, zaloMaxMessagePages)
		}
		if err := ctx.Err(); err != nil {
			return collected(), fmt.Errorf("%w: %w", ErrZaloMessageCoverageIncomplete, zaloSafe("message traversal cancelled", err))
		}

		body, err := z.doMessageRequest(ctx, conversationID, offset)
		if err != nil {
			return collected(), fmt.Errorf("%w: page %d: %w", ErrZaloMessageCoverageIncomplete, page+1, err)
		}
		rows, err := parseZaloMessagePage(body)
		if err != nil {
			return collected(), fmt.Errorf("%w (page %d)", err, page+1)
		}
		if err := ctx.Err(); err != nil {
			return collected(), fmt.Errorf("%w: %w", ErrZaloMessageCoverageIncomplete, zaloSafe("message traversal cancelled", err))
		}
		if len(rows) == 0 {
			return collected(), nil
		}

		// A page whose rows equal an earlier page means the offset is not moving through new
		// data. Only the canonical rows count: envelope text may change between repeats.
		h := sha256.New()
		for _, r := range rows {
			fmt.Fprintf(h, "%d:%s", len(r.key), r.key)
		}
		var fp [sha256.Size]byte
		copy(fp[:], h.Sum(nil))
		if seenPages[fp] {
			return collected(), fmt.Errorf("%w: page %d repeats an earlier page", ErrZaloMessageCoverageIncomplete, page+1)
		}
		seenPages[fp] = true

		for _, r := range rows {
			if existing, ok := byID[r.id]; ok {
				if existing.key != r.key {
					return collected(), fmt.Errorf("%w: page %d has conflicting rows for one message id", ErrZaloMessageCoverageIncomplete, page+1)
				}
				continue
			}
			byID[r.id] = r
			order = append(order, r.id)
		}
		offset += len(rows)
	}
}

func (z *ZaloOAAdapter) HealthCheck(ctx context.Context) error {
	_, err := z.doRequest(ctx, "GET", zaloAPIBaseV2+"/getoa", nil)
	return err
}

package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	fbGraphBase = "https://graph.facebook.com/v21.0"
	fbGraphHost = "graph.facebook.com"
	fbGraphPath = "/v21.0"

	// fbMaxConversationPages bounds one exhaustive enumeration. Reaching it before a terminal
	// page is incomplete coverage, never a truncated success (CCMAI-RUNTIME-022).
	fbMaxConversationPages = 500

	// Message traversal bounds (CCMAI-RUNTIME-031): reaching the page budget before a terminal
	// page is incomplete coverage; an oversized response is never terminal proof.
	fbMaxMessagePages         = 500
	fbMaxMessageResponseBytes = 8 << 20
)

// ErrFacebookCoverageIncomplete marks a conversation enumeration that did not observe a
// terminal page or could not trust a page. Rows returned with it are diagnostic only.
var ErrFacebookCoverageIncomplete = errors.New("facebook conversation coverage incomplete")

// ErrFacebookMessageCoverageIncomplete marks a message traversal that did not reach a trusted
// terminal page or met an untrusted page, row, link or budget limit (CCMAI-RUNTIME-031). Messages
// returned with it are diagnostic only. Distinct from the conversation identity above.
var ErrFacebookMessageCoverageIncomplete = errors.New("facebook message coverage incomplete")

// FacebookCredentials holds credentials for Facebook Graph API.
type FacebookCredentials struct {
	PageID      string `json:"page_id"`
	AccessToken string `json:"access_token"` // Long-lived page access token
}

type FacebookAdapter struct {
	creds  FacebookCredentials
	client *http.Client
}

func NewFacebookAdapter(creds FacebookCredentials) *FacebookAdapter {
	return &FacebookAdapter{
		creds:  creds,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *FacebookAdapter) doRequest(ctx context.Context, rawURL string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, errors.New("create facebook api request failed")
	}

	// Add access_token if not already in URL
	q := req.URL.Query()
	if q.Get("access_token") == "" {
		q.Set("access_token", f.creds.AccessToken)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := f.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("facebook api request failed: %w", ctx.Err())
		}
		// A transport's inner error can itself echo the token-bearing request URL.
		return nil, errors.New("facebook api request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("facebook api read body failed")
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("facebook api decode failed: %w", err)
	}

	if errObj, ok := result["error"].(map[string]interface{}); ok {
		code, _ := errObj["code"].(float64)
		// Graph's message is untrusted response content and can echo the token-bearing URL.
		return nil, fmt.Errorf("facebook api error: (#%.0f)", code)
	}

	return result, nil
}

// validNextURL accepts a pagination URL only if it stays on the Graph API conversations
// endpoint of this page over HTTPS. key is the query without the access token, used to detect
// a repeated cursor without ever keeping the token.
func (f *FacebookAdapter) validNextURL(raw string) (key string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" {
		return "", false
	}
	if u.Hostname() != fbGraphHost || (u.Port() != "" && u.Port() != "443") {
		return "", false
	}
	if u.Path != fbGraphPath+"/"+f.creds.PageID+"/conversations" {
		return "", false
	}
	q := u.Query()
	q.Del("access_token")
	return q.Encode(), true
}

// FetchRecentConversations enumerates every conversation updated at or after `since` by
// following all pages to a terminal one. A limit <= 0 means exhaustive; a positive limit
// that the window would exceed is an error, never a silent truncation. A page that cannot be
// trusted, a repeated cursor, a cancelled context or the page budget returns
// ErrFacebookCoverageIncomplete (or the request error) together with the rows seen so far,
// which are diagnostic only.
func (f *FacebookAdapter) FetchRecentConversations(ctx context.Context, since time.Time, limit int) ([]SyncedConversation, error) {
	var conversations []SyncedConversation
	seenIDs := map[string]bool{}
	seenPages := map[string]bool{}
	nextURL := fmt.Sprintf("%s/%s/conversations?fields=id,link,updated_time,participants&limit=100",
		fbGraphBase, f.creds.PageID)
	if key, ok := f.validNextURL(nextURL); ok {
		seenPages[key] = true
	}

	for pages := 0; ; pages++ {
		if pages >= fbMaxConversationPages {
			return conversations, fmt.Errorf("%w: page budget of %d reached before the last page", ErrFacebookCoverageIncomplete, fbMaxConversationPages)
		}
		if err := ctx.Err(); err != nil {
			return conversations, fmt.Errorf("%w: %v", ErrFacebookCoverageIncomplete, err)
		}

		result, err := f.doRequest(ctx, nextURL)
		if err != nil {
			return conversations, err
		}

		data, ok := result["data"].([]interface{})
		if !ok {
			return conversations, fmt.Errorf("%w: page %d has no data array", ErrFacebookCoverageIncomplete, pages+1)
		}

		for _, item := range data {
			conv, ok := item.(map[string]interface{})
			if !ok {
				return conversations, fmt.Errorf("%w: page %d has a malformed conversation", ErrFacebookCoverageIncomplete, pages+1)
			}
			convID, _ := conv["id"].(string)
			updStr, _ := conv["updated_time"].(string)
			updatedAt, err := time.Parse("2006-01-02T15:04:05-0700", updStr)
			if convID == "" || err != nil {
				return conversations, fmt.Errorf("%w: page %d has a conversation without a valid id or updated_time", ErrFacebookCoverageIncomplete, pages+1)
			}

			// Every row is examined: the ordering of pages is not relied on.
			if !since.IsZero() && updatedAt.Before(since) {
				continue
			}
			if seenIDs[convID] {
				continue
			}
			seenIDs[convID] = true

			// Extract participant name (the non-page user)
			customerName := ""
			if participants, ok := conv["participants"].(map[string]interface{}); ok {
				if pData, ok := participants["data"].([]interface{}); ok {
					for _, p := range pData {
						participant, _ := p.(map[string]interface{})
						pID, _ := participant["id"].(string)
						if pID != f.creds.PageID {
							customerName, _ = participant["name"].(string)
							break
						}
					}
				}
			}

			conversations = append(conversations, SyncedConversation{
				ExternalID:     convID,
				ExternalUserID: convID,
				CustomerName:   customerName,
				LastMessageAt:  updatedAt,
				Metadata:       conv,
			})
			if limit > 0 && len(conversations) > limit {
				return conversations[:limit], fmt.Errorf("%w: more than %d eligible conversations in the window", ErrFacebookCoverageIncomplete, limit)
			}
		}

		// Cursor-based pagination: an absent next link is the terminal page (an empty page with
		// a next link is not); anything else must be a safe, new Graph URL.
		pagingValue, hasPaging := result["paging"]
		if !hasPaging {
			return conversations, nil
		}
		paging, ok := pagingValue.(map[string]interface{})
		if !ok {
			return conversations, fmt.Errorf("%w: page %d has malformed paging", ErrFacebookCoverageIncomplete, pages+1)
		}
		nextValue, hasNext := paging["next"]
		if !hasNext {
			return conversations, nil
		}
		nextRaw, ok := nextValue.(string)
		if !ok || nextRaw == "" {
			return conversations, fmt.Errorf("%w: page %d has a malformed next link", ErrFacebookCoverageIncomplete, pages+1)
		}
		key, ok := f.validNextURL(nextRaw)
		if !ok {
			return conversations, fmt.Errorf("%w: page %d has an unsafe next link", ErrFacebookCoverageIncomplete, pages+1)
		}
		if seenPages[key] {
			return conversations, fmt.Errorf("%w: page %d repeats an earlier cursor", ErrFacebookCoverageIncomplete, pages+1)
		}
		seenPages[key] = true
		nextURL = nextRaw
	}
}

// doMessageRequest is the message-only request path (CCMAI-RUNTIME-031). Unlike the shared
// doRequest it blocks every redirect before following Location, rejects non-2xx responses,
// bounds the body at fbMaxMessageResponseBytes before decoding and never lets a transport or
// provider text reach the returned error. The injected transport and timeout are preserved by
// copying the client; the shared client and its redirect policy are not modified.
func (f *FacebookAdapter) doMessageRequest(ctx context.Context, rawURL string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.New("create facebook api request failed")
	}
	client := *f.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("facebook api request failed: %w", ctx.Err())
		}
		// A transport's inner error can itself echo the token-bearing request URL.
		return nil, errors.New("facebook api request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, fbMaxMessageResponseBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("facebook api read body failed: %w", ctx.Err())
		}
		return nil, errors.New("facebook api read body failed")
	}
	if len(body) > fbMaxMessageResponseBytes {
		return nil, errors.New("facebook api response exceeded size budget")
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("facebook api error: http %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.New("facebook api decode failed")
	}
	if errObj, ok := result["error"].(map[string]interface{}); ok {
		code, _ := errObj["code"].(float64)
		return nil, fmt.Errorf("facebook api error: (#%.0f)", code)
	}
	return result, nil
}

// messageEndpoint builds the canonical (token-free) message URL for a page query and returns
// its identity key. The URL is rebuilt from validated parts, never forwarded verbatim, and the
// configured token replaces any token the provider link carried.
func (f *FacebookAdapter) messageEndpoint(escapedPath string, query url.Values) (sendURL, key string) {
	clean := url.Values{}
	for k, vs := range query {
		if k != "access_token" {
			clean[k] = vs
		}
	}
	key = escapedPath + "?" + clean.Encode()
	clean.Set("access_token", f.creds.AccessToken)
	return "https://" + fbGraphHost + escapedPath + "?" + clean.Encode(), key
}

// validMessageNext accepts a pagination link only if it is an absolute HTTPS URL on the Graph
// host (no userinfo, fragment or non-default port) whose decoded path is exactly this
// conversation's v21.0 messages endpoint. It returns the link's query without validation of any
// token value; the caller rebuilds the outbound URL.
func validMessageNext(raw, wantPath string) (url.Values, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Opaque != "" || u.User != nil || u.Fragment != "" || u.RawFragment != "" {
		return nil, false
	}
	if u.Hostname() != fbGraphHost || (u.Port() != "" && u.Port() != "443") {
		return nil, false
	}
	if u.Path != wantPath {
		return nil, false
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, false
	}
	return q, true
}

// FetchMessages reads every message of a conversation with created_time >= since by following
// the paging.next chain to a terminal page (CCMAI-RUNTIME-031).
//
// Only a valid page with no paging, or paging without next, is terminal; an empty or duplicate-only
// page that carries a safe next link continues, and message IDs alone never prove a cursor did
// not progress. Every page needs a data array and every row a non-blank string id and a parseable
// non-zero created_time, checked before deduplication or since filtering; paging and next must be
// well formed even on an empty or old page. Next links must stay on the Graph host and this
// conversation's v21.0 messages path and are validated before any request; the configured token
// replaces whatever token a link carries, canonical link repeats fail, and traversal is capped at
// fbMaxMessagePages. since only filters output (inclusive); the result is deduplicated by ID
// (first mapping kept) and sorted ascending by time, ties in first-encounter order. Any failure
// returns the rows seen so far for diagnostics only. Nil-error means this local contract
// completed; it does not establish live visibility, retention or snapshot stability.
func (f *FacebookAdapter) FetchMessages(ctx context.Context, conversationID string, since time.Time) ([]SyncedMessage, error) {
	if strings.TrimSpace(conversationID) == "" {
		return nil, fmt.Errorf("%w: blank conversation id", ErrFacebookMessageCoverageIncomplete)
	}
	escapedPath := fbGraphPath + "/" + url.PathEscape(conversationID) + "/messages"
	wantPath := fbGraphPath + "/" + conversationID + "/messages"

	var messages []SyncedMessage
	seenIDs := map[string]bool{}
	seenPages := map[string]bool{}
	nextURL, key := f.messageEndpoint(escapedPath, url.Values{
		"fields": {"id,message,from,to,created_time,attachments,shares,sticker"},
		"limit":  {"100"},
	})
	seenPages[key] = true

	for page := 0; ; page++ {
		if page >= fbMaxMessagePages {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page budget of %d reached before the last page", ErrFacebookMessageCoverageIncomplete, fbMaxMessagePages)
		}
		if err := ctx.Err(); err != nil {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: %w", ErrFacebookMessageCoverageIncomplete, err)
		}

		result, err := f.doMessageRequest(ctx, nextURL)
		if err != nil {
			return sortMessagesBySentAt(messages), err
		}
		if err := ctx.Err(); err != nil {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: %w", ErrFacebookMessageCoverageIncomplete, err)
		}

		data, ok := result["data"].([]interface{})
		if !ok {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has no data array", ErrFacebookMessageCoverageIncomplete, page+1)
		}

		// Validate every row (old and repeated ones included) before dedupe or filtering.
		rows := make([]map[string]interface{}, len(data))
		times := make([]time.Time, len(data))
		for i, item := range data {
			row, ok := item.(map[string]interface{})
			if !ok {
				return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has a malformed message", ErrFacebookMessageCoverageIncomplete, page+1)
			}
			id, _ := row["id"].(string)
			ts, _ := row["created_time"].(string)
			sentAt, err := time.Parse("2006-01-02T15:04:05-0700", ts)
			if strings.TrimSpace(id) == "" || err != nil || sentAt.IsZero() {
				return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has a message without a valid id or created_time", ErrFacebookMessageCoverageIncomplete, page+1)
			}
			rows[i], times[i] = row, sentAt
		}
		for i, row := range rows {
			id, _ := row["id"].(string)
			if seenIDs[id] {
				continue
			}
			seenIDs[id] = true
			if !since.IsZero() && times[i].Before(since) {
				continue
			}
			messages = append(messages, f.toSyncedMessage(row, id, times[i]))
		}

		// Paging decides the terminal page: absent paging or no next is the end; anything else
		// must be a safe, new link. This holds for empty and older-than-since pages as well.
		pagingValue, hasPaging := result["paging"]
		if !hasPaging {
			return sortMessagesBySentAt(messages), nil
		}
		paging, ok := pagingValue.(map[string]interface{})
		if !ok {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has malformed paging", ErrFacebookMessageCoverageIncomplete, page+1)
		}
		nextValue, hasNext := paging["next"]
		if !hasNext {
			return sortMessagesBySentAt(messages), nil
		}
		nextRaw, ok := nextValue.(string)
		if !ok || nextRaw == "" {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has a malformed next link", ErrFacebookMessageCoverageIncomplete, page+1)
		}
		query, ok := validMessageNext(nextRaw, wantPath)
		if !ok {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has an unsafe next link", ErrFacebookMessageCoverageIncomplete, page+1)
		}
		var pageKey string
		nextURL, pageKey = f.messageEndpoint(escapedPath, query)
		if seenPages[pageKey] {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d repeats an earlier cursor", ErrFacebookMessageCoverageIncomplete, page+1)
		}
		seenPages[pageKey] = true
	}
}

// toSyncedMessage maps one validated Graph message row (mapping semantics unchanged by F02-E).
func (f *FacebookAdapter) toSyncedMessage(msg map[string]interface{}, msgID string, sentAt time.Time) SyncedMessage {
	content, _ := msg["message"].(string)

	// Determine sender type
	senderType := "customer"
	senderName := ""
	if from, ok := msg["from"].(map[string]interface{}); ok {
		fromID, _ := from["id"].(string)
		senderName, _ = from["name"].(string)
		if fromID == f.creds.PageID {
			senderType = "agent"
		}
	}

	syncedMsg := SyncedMessage{
		ExternalID:  msgID,
		SenderType:  senderType,
		SenderName:  senderName,
		Content:     content,
		ContentType: "text",
		SentAt:      sentAt,
		RawData:     msg,
	}

	// Parse attachments
	if attachData, ok := msg["attachments"].(map[string]interface{}); ok {
		if aData, ok := attachData["data"].([]interface{}); ok {
			for _, a := range aData {
				att, _ := a.(map[string]interface{})
				aType, _ := att["mime_type"].(string)
				aName, _ := att["name"].(string)
				aURL := ""
				if payload, ok := att["image_data"].(map[string]interface{}); ok {
					aURL, _ = payload["url"].(string)
				} else if payload, ok := att["video_data"].(map[string]interface{}); ok {
					aURL, _ = payload["url"].(string)
				} else if fileURL, ok := att["file_url"].(string); ok {
					aURL = fileURL
				}
				// Fallback: top-level url field
				if aURL == "" {
					if topURL, ok := att["url"].(string); ok {
						aURL = topURL
					}
				}
				// Fallback: media.image.src (StoryAttachment format)
				if aURL == "" {
					if media, ok := att["media"].(map[string]interface{}); ok {
						if img, ok := media["image"].(map[string]interface{}); ok {
							aURL, _ = img["src"].(string)
						}
					}
				}
				syncedMsg.Attachments = append(syncedMsg.Attachments, Attachment{
					Type: aType,
					URL:  aURL,
					Name: aName,
				})
			}
			if len(syncedMsg.Attachments) > 0 {
				syncedMsg.ContentType = "attachment"
			}
		}
	}

	// Sticker
	if _, ok := msg["sticker"].(string); ok {
		syncedMsg.ContentType = "sticker"
	}
	return syncedMsg
}

func (f *FacebookAdapter) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/%s?fields=id,name", fbGraphBase, f.creds.PageID)
	_, err := f.doRequest(ctx, healthURL)
	return err
}

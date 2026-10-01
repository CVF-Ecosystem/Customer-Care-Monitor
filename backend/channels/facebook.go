package channels

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	fbGraphBase = "https://graph.facebook.com/v21.0"
	fbGraphHost = "graph.facebook.com"
	fbGraphPath = "/v21.0"

	// fbMaxConversationPages bounds one exhaustive enumeration. Reaching it before a terminal
	// page is incomplete coverage, never a truncated success (CCMAI-RUNTIME-022).
	fbMaxConversationPages = 500
)

// ErrFacebookCoverageIncomplete marks a conversation enumeration that did not observe a
// terminal page or could not trust a page. Rows returned with it are diagnostic only.
var ErrFacebookCoverageIncomplete = errors.New("facebook conversation coverage incomplete")

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

func (f *FacebookAdapter) FetchMessages(ctx context.Context, conversationID string, since time.Time) ([]SyncedMessage, error) {
	var messages []SyncedMessage
	nextURL := fmt.Sprintf("%s/%s/messages?fields=id,message,from,to,created_time,attachments,shares,sticker&limit=100",
		fbGraphBase, conversationID)

	for nextURL != "" {
		result, err := f.doRequest(ctx, nextURL)
		if err != nil {
			return messages, err
		}

		data, ok := result["data"].([]interface{})
		if !ok || len(data) == 0 {
			break
		}

		for _, item := range data {
			msg, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			var sentAt time.Time
			if ts, ok := msg["created_time"].(string); ok {
				sentAt, _ = time.Parse("2006-01-02T15:04:05-0700", ts)
			}

			if !since.IsZero() && sentAt.Before(since) {
				return messages, nil
			}

			msgID, _ := msg["id"].(string)
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

			messages = append(messages, syncedMsg)
		}

		// Cursor pagination
		nextURL = ""
		if paging, ok := result["paging"].(map[string]interface{}); ok {
			if next, ok := paging["next"].(string); ok {
				nextURL = next
			}
		}
	}

	return messages, nil
}

func (f *FacebookAdapter) HealthCheck(ctx context.Context) error {
	healthURL := fmt.Sprintf("%s/%s?fields=id,name", fbGraphBase, f.creds.PageID)
	_, err := f.doRequest(ctx, healthURL)
	return err
}

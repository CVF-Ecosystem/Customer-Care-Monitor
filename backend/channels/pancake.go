package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Pancake (pages.fm) gom nhiều nền tảng chat — Facebook, Instagram, Zalo OA,
// TikTok, Shopee… — về một chỗ, nên một adapter đọc được hết các page mà
// khách đã kết nối vào Pancake. Tài liệu: https://developer.pancake.biz/
const (
	pancakePublicV1 = "https://pages.fm/api/public_api/v1"
	pancakePublicV2 = "https://pages.fm/api/public_api/v2"

	// Pancake trả tối đa 60 cuộc chat và 30 tin mỗi lần gọi.
	pancakeConversationPageSize = 60

	// Pancake giới hạn 5 lượt gọi mỗi giây cho mỗi page. Giữ khoảng cách 250ms
	// (4 lượt/giây) để còn dư chỗ cho lượt gọi của chính Pancake và bên khác.
	pancakeMinRequestInterval = 250 * time.Millisecond
	pancakeMaxRetries         = 3
	pancakeRetryBackoff       = time.Second

	// Chặn vòng lặp phân trang nếu API trả dữ liệu bất thường.
	pancakeMaxPages = 200
	// Bound response memory before decoding; an oversized response is never terminal proof.
	pancakeMaxResponseBytes = 8 << 20
)

// ErrPancakeCoverageIncomplete đánh dấu lần duyệt danh sách hội thoại chưa thấy trang rỗng cuối
// cùng hoặc gặp trang không tin được. Các hội thoại trả kèm lỗi này chỉ để chẩn đoán, không phải
// một cửa sổ đã phủ đủ (CCMAI-RUNTIME-023).
var ErrPancakeCoverageIncomplete = errors.New("pancake conversation coverage incomplete")

// ErrPancakeMessageCoverageIncomplete đánh dấu lần duyệt tin nhắn của một hội thoại chưa thấy
// trang rỗng cuối cùng hoặc gặp dữ liệu không tin được (CCMAI-RUNTIME-030). Các tin trả kèm lỗi này
// chỉ để chẩn đoán, không phải một cửa sổ tin nhắn đã phủ đủ. Tách biệt với lỗi hội thoại ở trên.
var ErrPancakeMessageCoverageIncomplete = errors.New("pancake message coverage incomplete")

// PancakeCredentials là thông tin để đọc một page qua API công khai của Pancake.
// Page Access Token lấy trong Pancake: Cài đặt page → Công cụ. Token không hết
// hạn cho tới khi admin tạo token mới.
type PancakeCredentials struct {
	PageID          string `json:"page_id"`
	PageAccessToken string `json:"page_access_token"`
}

type PancakeAdapter struct {
	creds  PancakeCredentials
	client *http.Client
	v1Base string
	v2Base string

	// Giữ nhịp gọi API để không vượt giới hạn của Pancake.
	mu          sync.Mutex
	lastRequest time.Time
	minInterval time.Duration
	backoff     time.Duration
}

func NewPancakeAdapter(creds PancakeCredentials) *PancakeAdapter {
	return &PancakeAdapter{
		creds:       creds,
		client:      &http.Client{Timeout: 30 * time.Second},
		v1Base:      pancakePublicV1,
		v2Base:      pancakePublicV2,
		minInterval: pancakeMinRequestInterval,
		backoff:     pancakeRetryBackoff,
	}
}

// waitTurn chờ tới lượt gọi tiếp theo theo nhịp minInterval.
func (p *PancakeAdapter) waitTurn(ctx context.Context) error {
	p.mu.Lock()
	wait := p.minInterval - time.Since(p.lastRequest)
	if wait < 0 {
		wait = 0
	}
	p.lastRequest = time.Now().Add(wait)
	p.mu.Unlock()

	if wait == 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(wait):
		return nil
	}
}

// doRequest gọi một endpoint GET của Pancake và giải mã JSON vào out.
// Pancake báo lỗi bằng HTTP 200 kèm "success": false, nên phải xem cả thân trả về.
func (p *PancakeAdapter) doRequest(ctx context.Context, endpoint string, params url.Values, out interface{}) error {
	if params == nil {
		params = url.Values{}
	}
	params.Set("page_access_token", p.creds.PageAccessToken)
	fullURL := endpoint + "?" + params.Encode()

	for attempt := 0; ; attempt++ {
		if err := p.waitTurn(ctx); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return errors.New("create pancake api request failed")
		}
		resp, err := p.client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("pancake api request failed: %w", ctx.Err())
			}
			// Transport causes can themselves contain the token-bearing URL.
			return errors.New("pancake api request failed")
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, pancakeMaxResponseBytes+1))
		resp.Body.Close()
		if err != nil {
			return errors.New("pancake api read body failed")
		}
		if len(body) > pancakeMaxResponseBytes {
			return errors.New("pancake api response exceeded size budget")
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < pancakeMaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.backoff * time.Duration(attempt+1)):
			}
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return fmt.Errorf("pancake api error: http %d", resp.StatusCode)
		}

		var status struct {
			Success   *bool `json:"success"`
			ErrorCode int   `json:"error_code"`
		}
		if err := json.Unmarshal(body, &status); err != nil {
			return fmt.Errorf("pancake api decode failed: %w", err)
		}
		if status.Success != nil && !*status.Success {
			// Provider text can echo URLs, encoded credentials or response content.
			return fmt.Errorf("pancake api error: (#%d)", status.ErrorCode)
		}
		if err := json.Unmarshal(body, out); err != nil {
			return fmt.Errorf("pancake api decode failed: %w", err)
		}
		return nil
	}
}

type pancakeSender struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	UID         string `json:"uid"`
	AdminName   string `json:"admin_name"`
	IsAutomated *bool  `json:"is_automated"`
	AIGenerated *bool  `json:"ai_generated"`
}

type pancakeConversation struct {
	ID           string        `json:"id"`
	Type         string        `json:"type"`
	PageID       string        `json:"page_id"`
	MessageCount int           `json:"message_count"`
	UpdatedAt    string        `json:"updated_at"`
	InsertedAt   string        `json:"inserted_at"`
	From         pancakeSender `json:"from"`
	Tags         []interface{} `json:"tags"`
}

type pancakeAttachment struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	URL       string `json:"url"`
	FileURL   string `json:"file_url"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	VideoData *struct {
		URL string `json:"url"`
	} `json:"video_data"`
}

type pancakeMessage struct {
	ID              string              `json:"id"`
	Type            string              `json:"type"`
	PageID          string              `json:"page_id"`
	Message         string              `json:"message"`
	OriginalMessage string              `json:"original_message"`
	From            pancakeSender       `json:"from"`
	InsertedAt      string              `json:"inserted_at"`
	IsRemoved       bool                `json:"is_removed"`
	Attachments     []pancakeAttachment `json:"attachments"`
}

// FetchRecentConversations duyệt hết các hội thoại INBOX có updated_at >= since (CCMAI-RUNTIME-023).
//
// until được chốt một lần trước trang đầu và dùng lại cho mọi trang. Trang sau đi theo ID của
// dòng cuối cùng trong trang trước (kể cả dòng COMMENT bị lọc). Tài liệu Pancake không bảo đảm
// trang ngắn là trang cuối, nên chỉ một mảng conversations rỗng mới kết thúc thành công. Mọi
// dòng đều được xét, không dựa vào thứ tự; dòng trùng ID chỉ lấy lần đầu. limit <= 0 là duyệt
// hết; limit > 0 mà cửa sổ còn nhiều hơn thì trả lỗi chứ không cắt ngắn. Trang thiếu/sai
// conversations, dòng thiếu ID hoặc updated_at hợp lệ, cursor lặp, lỗi trang, ctx bị huỷ hay hết
// ngân sách trang đều trả ErrPancakeCoverageIncomplete (hoặc lỗi gọi API) kèm các dòng đã thấy,
// chỉ để chẩn đoán.
func (p *PancakeAdapter) FetchRecentConversations(ctx context.Context, since time.Time, limit int) ([]SyncedConversation, error) {
	var conversations []SyncedConversation
	endpoint := fmt.Sprintf("%s/pages/%s/conversations", p.v2Base, url.PathEscape(p.creds.PageID))
	seenIDs := make(map[string]bool)
	seenCursors := make(map[string]bool)
	lastID := ""

	var sinceParam string
	untilParam := strconv.FormatInt(time.Now().Unix(), 10)
	if !since.IsZero() {
		sinceParam = strconv.FormatInt(since.Unix(), 10)
	}

	for page := 0; ; page++ {
		if page >= pancakeMaxPages {
			return conversations, fmt.Errorf("%w: page budget of %d reached before an empty page", ErrPancakeCoverageIncomplete, pancakeMaxPages)
		}
		if err := ctx.Err(); err != nil {
			return conversations, fmt.Errorf("%w: %v", ErrPancakeCoverageIncomplete, err)
		}

		params := url.Values{}
		params.Set("type", "INBOX")
		params.Set("order_by", "updated_at")
		params.Set("until", untilParam)
		if sinceParam != "" {
			params.Set("since", sinceParam)
		}
		if lastID != "" {
			params.Set("last_conversation_id", lastID)
		}

		var result struct {
			Conversations json.RawMessage `json:"conversations"`
		}
		if err := p.doRequest(ctx, endpoint, params, &result); err != nil {
			return conversations, err
		}
		if err := ctx.Err(); err != nil {
			return conversations, fmt.Errorf("%w: %v", ErrPancakeCoverageIncomplete, err)
		}
		raw := bytes.TrimSpace(result.Conversations)
		if len(raw) == 0 || raw[0] != '[' {
			return conversations, fmt.Errorf("%w: page %d has no conversations array", ErrPancakeCoverageIncomplete, page+1)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return conversations, fmt.Errorf("%w: page %d has a malformed conversations array", ErrPancakeCoverageIncomplete, page+1)
		}
		if len(rows) == 0 {
			return conversations, nil // trang rỗng: hết chuỗi cursor
		}

		for _, row := range rows {
			var conv pancakeConversation
			if err := json.Unmarshal(row, &conv); err != nil {
				return conversations, fmt.Errorf("%w: page %d has a malformed conversation", ErrPancakeCoverageIncomplete, page+1)
			}
			updatedAt := parsePancakeTime(conv.UpdatedAt)
			if conv.ID == "" || updatedAt.IsZero() {
				return conversations, fmt.Errorf("%w: page %d has a conversation without a valid id or updated_at", ErrPancakeCoverageIncomplete, page+1)
			}
			// Chỉ lấy hội thoại tin nhắn; bình luận dưới bài viết không thuộc
			// phạm vi đánh giá CSKH, kể cả khi API bỏ qua bộ lọc type.
			if conv.Type != "" && conv.Type != "INBOX" {
				continue
			}
			if !since.IsZero() && updatedAt.Before(since) {
				continue
			}
			if seenIDs[conv.ID] {
				continue
			}
			seenIDs[conv.ID] = true
			conversations = append(conversations, SyncedConversation{
				ExternalID:     conv.ID,
				ExternalUserID: conv.From.ID,
				CustomerName:   conv.From.Name,
				LastMessageAt:  updatedAt,
				// Chỉ giữ vài trường cần thiết. Bản gốc có số điện thoại, đơn
				// hàng của khách — không lưu những thứ đó.
				Metadata: map[string]interface{}{
					"type":          conv.Type,
					"message_count": conv.MessageCount,
					"inserted_at":   conv.InsertedAt,
					"updated_at":    conv.UpdatedAt,
					"tags":          conv.Tags,
				},
			})
			if limit > 0 && len(conversations) > limit {
				return conversations[:limit], fmt.Errorf("%w: more than %d eligible conversations in the window", ErrPancakeCoverageIncomplete, limit)
			}
		}

		// Cursor là ID của dòng vật lý cuối trang, kể cả dòng đã bị lọc.
		var last pancakeConversation
		_ = json.Unmarshal(rows[len(rows)-1], &last) // đã kiểm tra hợp lệ ở vòng trên
		if last.ID == lastID || seenCursors[last.ID] {
			return conversations, fmt.Errorf("%w: page %d repeats an earlier cursor", ErrPancakeCoverageIncomplete, page+1)
		}
		seenCursors[last.ID] = true
		lastID = last.ID
	}
}

// FetchMessages đọc toàn bộ tin nhắn của một hội thoại có inserted_at >= since (CCMAI-RUNTIME-030).
//
// Điểm đầu không gửi current_count; các trang sau gửi số dòng vật lý đã đọc (kể cả dòng trùng và
// dòng cũ hơn since). Tài liệu Pancake không bảo đảm trang ngắn là trang cuối, không cho biết thứ
// tự giữa các trang và không có snapshot, nên: chỉ một mảng messages rỗng hợp lệ mới kết thúc
// thành công; since chỉ lọc kết quả (gồm cả mốc bằng), không dừng sớm. Mọi dòng được kiểm tra
// (id chuỗi không rỗng, inserted_at hợp lệ) trước khi lọc/khử trùng. Trang không rỗng mà không có
// ID mới (lặp lại), trang thiếu/sai mảng messages, dòng hỏng, ctx bị huỷ hay hết ngân sách trang
// đều trả ErrPancakeMessageCoverageIncomplete (hoặc lỗi gọi API) kèm các tin đã thấy, chỉ để chẩn
// đoán. Kết quả khử trùng theo ID (giữ bản đầu tiên) và xếp tăng dần theo SentAt, bằng nhau thì
// giữ thứ tự gặp đầu tiên. Nil-error chỉ nghĩa là duyệt cục bộ này hoàn tất; không chứng minh
// lịch sử không đổi trong lúc đọc.
func (p *PancakeAdapter) FetchMessages(ctx context.Context, conversationID string, since time.Time) ([]SyncedMessage, error) {
	endpoint := fmt.Sprintf("%s/pages/%s/conversations/%s/messages",
		p.v1Base, url.PathEscape(p.creds.PageID), url.PathEscape(conversationID))

	var messages []SyncedMessage
	seen := make(map[string]bool)
	consumed := 0 // số dòng vật lý đã đọc = current_count của trang kế tiếp

	for page := 0; ; page++ {
		if page >= pancakeMaxPages {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page budget of %d reached before an empty page", ErrPancakeMessageCoverageIncomplete, pancakeMaxPages)
		}
		if err := ctx.Err(); err != nil {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: %w", ErrPancakeMessageCoverageIncomplete, err)
		}

		params := url.Values{}
		if consumed > 0 {
			params.Set("current_count", strconv.Itoa(consumed))
		}

		var result struct {
			Messages json.RawMessage `json:"messages"`
		}
		if err := p.doRequest(ctx, endpoint, params, &result); err != nil {
			return sortMessagesBySentAt(messages), err
		}
		if err := ctx.Err(); err != nil {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: %w", ErrPancakeMessageCoverageIncomplete, err)
		}
		raw := bytes.TrimSpace(result.Messages)
		if len(raw) == 0 || raw[0] != '[' {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has no messages array", ErrPancakeMessageCoverageIncomplete, page+1)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has a malformed messages array", ErrPancakeMessageCoverageIncomplete, page+1)
		}
		if len(rows) == 0 {
			return sortMessagesBySentAt(messages), nil // trang rỗng: hết lịch sử theo offset
		}

		// Kiểm tra từng dòng (kể cả dòng cũ hơn since hoặc trùng ID) trước khi lọc/khử trùng.
		parsed := make([]pancakeMessage, len(rows))
		for i, row := range rows {
			if err := json.Unmarshal(row, &parsed[i]); err != nil {
				return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has a malformed message", ErrPancakeMessageCoverageIncomplete, page+1)
			}
			if strings.TrimSpace(parsed[i].ID) == "" || parsePancakeTime(parsed[i].InsertedAt).IsZero() {
				return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d has a message without a valid id or inserted_at", ErrPancakeMessageCoverageIncomplete, page+1)
			}
		}

		newCount := 0
		for _, m := range parsed {
			consumed++
			if seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			newCount++

			msg := p.toSyncedMessage(m)
			if !since.IsZero() && msg.SentAt.Before(since) {
				continue
			}
			messages = append(messages, msg)
		}
		if newCount == 0 {
			return sortMessagesBySentAt(messages), fmt.Errorf("%w: page %d repeats only messages already seen", ErrPancakeMessageCoverageIncomplete, page+1)
		}
	}
}

// sortMessagesBySentAt xếp tăng dần theo thời gian gửi; bằng nhau thì giữ thứ tự gặp đầu tiên.
func sortMessagesBySentAt(msgs []SyncedMessage) []SyncedMessage {
	sort.SliceStable(msgs, func(i, j int) bool { return msgs[i].SentAt.Before(msgs[j].SentAt) })
	return msgs
}

func (p *PancakeAdapter) toSyncedMessage(m pancakeMessage) SyncedMessage {
	senderType := "customer"
	senderName := m.From.Name
	pageID := m.PageID
	if pageID == "" {
		pageID = p.creds.PageID
	}
	if m.From.ID != "" && m.From.ID == pageID {
		senderType = "agent"
		// Pancake cho biết nhân viên nào trả lời; tên page thì ai cũng như nhau.
		if m.From.AdminName != "" {
			senderName = m.From.AdminName
		}
	}

	msg := SyncedMessage{
		ExternalID:  m.ID,
		SenderType:  senderType,
		SenderName:  senderName,
		Content:     pancakeContent(m),
		ContentType: "text",
		SentAt:      parsePancakeTime(m.InsertedAt),
	}

	isSticker := false
	for _, a := range m.Attachments {
		att, ok := toAttachment(a)
		if !ok {
			continue
		}
		if a.Type == "sticker" {
			isSticker = true
		}
		msg.Attachments = append(msg.Attachments, att)
	}
	if len(msg.Attachments) > 0 {
		msg.ContentType = "attachment"
	}
	if isSticker {
		msg.ContentType = "sticker"
	}

	// Giữ bản rút gọn của tin gốc để tra lại khi cần. Bỏ email và các thông
	// tin cá nhân khác của khách mà Pancake trả kèm.
	raw := map[string]interface{}{
		"id":          m.ID,
		"type":        m.Type,
		"inserted_at": m.InsertedAt,
		"from": map[string]interface{}{
			"id":         m.From.ID,
			"name":       m.From.Name,
			"admin_name": m.From.AdminName,
			"uid":        m.From.UID,
		},
		"attachments": m.Attachments,
	}
	if m.From.IsAutomated != nil {
		raw["is_automated"] = *m.From.IsAutomated
	}
	if m.From.AIGenerated != nil {
		raw["ai_generated"] = *m.From.AIGenerated
	}
	if m.IsRemoved {
		raw["is_removed"] = true
	}
	msg.RawData = raw

	return msg
}

// toAttachment chuyển một đính kèm của Pancake về dạng chung.
//
// Ảnh và sticker nằm trên máy chủ của Pancake. Video và file nằm trên máy chủ
// Facebook với link có hạn khoảng hai ngày, nên muốn xem lại lâu dài phải bật
// lưu file để lúc đồng bộ tải về luôn.
func toAttachment(a pancakeAttachment) (Attachment, bool) {
	attURL := a.URL
	switch {
	case a.FileURL != "":
		// File không có trường type, link nằm ở file_url.
		attURL = a.FileURL
	case a.VideoData != nil && a.VideoData.URL != "":
		// url của video chỉ là ảnh đại diện; video thật ở video_data.url.
		attURL = a.VideoData.URL
	}
	if attURL == "" {
		return Attachment{}, false
	}

	attType := a.MimeType
	if attType == "" {
		attType = a.Type
	}
	if attType == "" {
		attType = "file"
	}

	name := a.Name
	if name == "" {
		// Không có tên thì lấy tên file trong link (thường là mã băm, không trùng).
		if u, err := url.Parse(attURL); err == nil {
			name = path.Base(u.Path)
		}
	}

	return Attachment{Type: attType, URL: attURL, Name: name}, true
}

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// pancakeContent lấy nội dung chữ của tin. original_message là chữ gốc; trường
// message đã bọc HTML (<div>…</div>) và mã hoá ký tự (&lt;3), chỉ dùng khi
// original_message rỗng.
func pancakeContent(m pancakeMessage) string {
	if m.OriginalMessage != "" {
		return m.OriginalMessage
	}
	text := htmlTagPattern.ReplaceAllString(m.Message, "")
	return strings.TrimSpace(html.UnescapeString(text))
}

// parsePancakeTime đọc thời gian Pancake trả về, ví dụ "2026-09-24T15:18:30.428000".
// Pancake dùng giờ UTC nhưng không ghi múi giờ.
func parsePancakeTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05.999999999", s, time.UTC); err == nil {
		return t
	}
	return time.Time{}
}

func (p *PancakeAdapter) HealthCheck(ctx context.Context) error {
	// Danh sách thẻ là endpoint nhẹ nhất cần page token.
	endpoint := fmt.Sprintf("%s/pages/%s/tags", p.v1Base, url.PathEscape(p.creds.PageID))
	var result map[string]interface{}
	return p.doRequest(ctx, endpoint, nil, &result)
}

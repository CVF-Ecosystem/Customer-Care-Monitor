package channels

// Offline Pancake proof harness (CCMAI-RUNTIME-034).
//
// The harness drives the unchanged PancakeAdapter through an explicitly injected
// http.RoundTripper, bounds every attempt/time/response, reconciles an independently
// authored inventory and emits a sanitized, deterministic receipt. It never discovers
// credentials or configuration, never falls back to http.DefaultTransport and never
// persists raw captures. R034 callers supply synthetic values only; the receipt is
// SYNTHETIC_OFFLINE evidence, not live-channel or AI-governance proof.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ProofSchemaVersion    = "pancake-proof-receipt/1"
	ProofEvidenceType     = "SYNTHETIC_OFFLINE"
	ProofMaxAttempts      = 50
	ProofMaxDuration      = 10 * time.Minute
	ProofMaxResponseBytes = 8 << 20
	proofRuns             = 2
	proofHost             = "pages.fm"
	proofConvTemplate     = "/api/public_api/v2/pages/{page}/conversations"
	proofMsgTemplate      = "/api/public_api/v1/pages/{page}/conversations/{conversation}/messages"
	proofTranscriptCeil   = 500
	proofBlockSafety      = 2 * time.Second
)

// ProofDisposition is the final (or per-run) outcome class.
type ProofDisposition string

const (
	ProofPass       ProofDisposition = "PASS"
	ProofFail       ProofDisposition = "FAIL"
	ProofIncomplete ProofDisposition = "INCOMPLETE"
	proofNotRun     ProofDisposition = "NOT_RUN"
)

// ErrPancakeProofInput marks rejected setup. Messages carry only a fixed reason code.
var ErrPancakeProofInput = errors.New("pancake proof input rejected")

// ErrPancakeProofUnsafe means the serialized receipt contained a raw sensitive value.
var ErrPancakeProofUnsafe = errors.New("pancake proof receipt failed sanitation")

var (
	errProofDenied   = errors.New("proof request denied")
	errProofBudget   = errors.New("proof budget exhausted")
	errProofRedirect = errors.New("proof redirect rejected")
	errProofOversize = errors.New("proof response oversize")
	errProofInner    = errors.New("proof transport failed")
	errProofRead     = errors.New("proof response read failed")
)

var (
	proofSHAPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	proofLabelPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)
	proofDigits       = regexp.MustCompile(`^[0-9]{1,12}$`)
)

// proofSafeSegment reports whether v is safe to place in one URL path segment even after any
// decoding: no dot segments, separators, escapes, query/fragment markers or control characters.
func proofSafeSegment(v string) bool {
	if v == "" || v == "." || v == ".." {
		return false
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f || r == ' ' || strings.ContainsRune(`/\%?#;:`, r) {
			return false
		}
	}
	return true
}

func proofInputErr(code string) error { return fmt.Errorf("%w: %s", ErrPancakeProofInput, code) }

// ---------------------------------------------------------------------------
// Inventory (independently authored expectations) and options

type ProofAttachment struct {
	Type string
	Name string
}

type ProofMessage struct {
	ID          string
	SentAt      time.Time
	SenderType  string
	ContentType string
	Attachments []ProofAttachment
}

type ProofConversation struct {
	ID        string
	UpdatedAt time.Time
	Messages  []ProofMessage
}

// ProofInventory lists only rows that MUST be observed. Old, duplicate and non-INBOX rows are
// expected to be filtered by the adapter and therefore do not appear here.
type ProofInventory struct {
	Provenance    string // short label, [A-Za-z0-9._:-]{1,64}
	PageID        string
	Conversations []ProofConversation
}

type PancakeProofOptions struct {
	SourceSHA    string // explicit 40-hex source revision
	PageID       string
	Token        string // synthetic in R034; never discovered
	PseudonymKey []byte
	Since        time.Time
	Transport    http.RoundTripper // required; no default
	MaxAttempts  int               // 1..ProofMaxAttempts, shared by both runs
	MaxDuration  time.Duration     // >0..ProofMaxDuration, shared by both runs
	// MinRequestInterval and RetryBackoff of 0 disable pacing/backoff (offline fixtures). A later
	// authorized live caller must set them explicitly.
	MinRequestInterval time.Duration
	RetryBackoff       time.Duration
	Now                func() time.Time
}

func (o *PancakeProofOptions) validate(inv *ProofInventory) error {
	switch {
	case o.Transport == nil:
		return proofInputErr("transport_required")
	case !proofSHAPattern.MatchString(o.SourceSHA):
		return proofInputErr("source_sha")
	case o.PageID == "" || o.Token == "":
		return proofInputErr("page_or_token_missing")
	case !proofSafeSegment(o.PageID):
		return proofInputErr("unsafe_identifier")
	case len(o.PseudonymKey) == 0:
		return proofInputErr("pseudonym_key_missing")
	case o.Since.IsZero():
		return proofInputErr("since_missing")
	case o.MaxAttempts < 1 || o.MaxAttempts > ProofMaxAttempts:
		return proofInputErr("max_attempts")
	case o.MaxDuration <= 0 || o.MaxDuration > ProofMaxDuration:
		return proofInputErr("max_duration")
	case o.MinRequestInterval < 0 || o.RetryBackoff < 0:
		return proofInputErr("negative_pacing")
	case inv == nil || !proofLabelPattern.MatchString(inv.Provenance):
		return proofInputErr("inventory_provenance")
	case inv.PageID != o.PageID:
		return proofInputErr("inventory_page_mismatch")
	}
	seenConv := map[string]bool{}
	for _, c := range inv.Conversations {
		if !proofSafeSegment(c.ID) {
			return proofInputErr("unsafe_identifier")
		}
		if c.UpdatedAt.IsZero() || seenConv[c.ID] {
			return proofInputErr("inventory_conversation")
		}
		seenConv[c.ID] = true
		seenMsg := map[string]bool{}
		for _, m := range c.Messages {
			if m.ID == "" || m.SentAt.IsZero() || seenMsg[m.ID] {
				return proofInputErr("inventory_message")
			}
			seenMsg[m.ID] = true
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Receipt

type ProofReceipt struct {
	SchemaVersion string           `json:"schema_version"`
	EvidenceType  string           `json:"evidence_type"`
	Live          bool             `json:"live"`
	Governance    bool             `json:"governance_claim"`
	SourceSHA     string           `json:"source_sha"`
	StartedAt     string           `json:"started_at"`
	FinishedAt    string           `json:"finished_at"`
	Inventory     ProofInvSummary  `json:"inventory"`
	Options       ProofOptSummary  `json:"options"`
	Budget        ProofBudget      `json:"budget"`
	Denials       []ProofDenial    `json:"denials"`
	Runs          []ProofRunReport `json:"runs"`
	Disposition   ProofDisposition `json:"disposition"`
	Reasons       []string         `json:"reasons"`
}

type ProofInvSummary struct {
	Digest            string `json:"digest"`
	Provenance        string `json:"provenance"`
	PageRef           string `json:"page_ref"`
	Since             string `json:"since"`
	ConversationCount int    `json:"conversation_count"`
	MessageCount      int    `json:"message_count"`
}

type ProofOptSummary struct {
	MaxAttempts        int   `json:"max_attempts"`
	MaxDurationMS      int64 `json:"max_duration_ms"`
	MinRequestInterval int64 `json:"min_request_interval_ms"`
	RetryBackoffMS     int64 `json:"retry_backoff_ms"`
	ResponseCeiling    int   `json:"response_ceiling_bytes"`
	Runs               int   `json:"runs"`
}

type ProofBudget struct {
	AttemptsUsed int   `json:"attempts_used"`
	Denied       int   `json:"denied"`
	ElapsedMS    int64 `json:"elapsed_ms"`
}

type ProofDenial struct {
	Run    int    `json:"run"`
	Reason string `json:"reason"`
}

type ProofRequestRecord struct {
	Seq             int    `json:"seq"`
	Run             int    `json:"run"`
	Kind            string `json:"kind"`
	Template        string `json:"endpoint_template"`
	ConversationRef string `json:"conversation_ref,omitempty"`
	CursorRef       string `json:"cursor_ref,omitempty"`
	CurrentCount    int    `json:"current_count,omitempty"`
	Status          int    `json:"status"`
	Outcome         string `json:"outcome"`
	Rows            int    `json:"rows"`
	EmptyTerminal   bool   `json:"empty_terminal"`
	ResponseBytes   int    `json:"response_bytes"`
}

type ProofRowCounts struct {
	PhysicalRows int `json:"physical_rows"`
	DuplicateRow int `json:"duplicate_rows"`
	OldRows      int `json:"old_rows"`
	NonInboxRows int `json:"non_inbox_rows,omitempty"`
	RawEligible  int `json:"raw_eligible"`
	Mapped       int `json:"mapped"`
}

type ProofMismatch struct {
	Ref    string   `json:"ref"`
	Fields []string `json:"fields"`
}

type ProofSetDiff struct {
	Expected       int             `json:"expected"`
	Observed       int             `json:"observed"`
	Missing        []string        `json:"missing"`
	Extra          []string        `json:"extra"`
	Mismapped      []ProofMismatch `json:"mismapped"`
	AdapterOmitted []string        `json:"adapter_omitted"`
}

type ProofRunReport struct {
	Index             int                  `json:"index"`
	StartedAt         string               `json:"started_at"`
	FinishedAt        string               `json:"finished_at"`
	ConversationUntil int64                `json:"conversation_until"`
	UntilStable       bool                 `json:"until_stable"`
	Requests          []ProofRequestRecord `json:"requests"`
	ConversationRows  ProofRowCounts       `json:"conversation_rows"`
	MessageRows       ProofRowCounts       `json:"message_rows"`
	ConversationTerm  bool                 `json:"conversation_terminal"`
	MessageTerminals  int                  `json:"message_terminals"`
	MessageFetches    int                  `json:"message_fetches"`
	Conversations     ProofSetDiff         `json:"conversations"`
	Messages          ProofSetDiff         `json:"messages"`
	Disposition       ProofDisposition     `json:"disposition"`
	Reasons           []string             `json:"reasons"`
}

// MarshalProofReceipt validates and serializes the receipt deterministically. It fails closed
// (ErrPancakeProofUnsafe, never echoing the rejected value) for unsupported schema, header, field
// classes or dispositions, and when any decoded string value or key contains a value from
// sensitive (values shorter than 6 bytes are protected by construction instead, since they would
// match arbitrary substrings).
func MarshalProofReceipt(r *ProofReceipt, sensitive []string) ([]byte, error) {
	if !validProofReceipt(r) {
		return nil, ErrPancakeProofUnsafe
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil || proofContainsSensitive(b, sensitive) {
		return nil, ErrPancakeProofUnsafe
	}
	return append(b, 10), nil
}

// proofContainsSensitive decodes the JSON document and compares semantic string values and keys,
// so JSON escaping of quotes, backslashes, newlines or HTML characters cannot hide a value.
// Undecodable input is treated as sensitive (fail closed).
func proofContainsSensitive(doc []byte, sensitive []string) bool {
	var v interface{}
	if json.Unmarshal(doc, &v) != nil {
		return true
	}
	var walk func(x interface{}) bool
	hit := func(str string) bool {
		for _, s := range sensitive {
			if len(s) >= 6 && strings.Contains(str, s) {
				return true
			}
		}
		return false
	}
	walk = func(x interface{}) bool {
		switch t := x.(type) {
		case string:
			return hit(t)
		case []interface{}:
			for _, e := range t {
				if walk(e) {
					return true
				}
			}
		case map[string]interface{}:
			for k, e := range t {
				if hit(k) || walk(e) {
					return true
				}
			}
		}
		return false
	}
	return walk(v)
}

var (
	proofRefPattern     = regexp.MustCompile(`^(pag|con|mes|cur)_[0-9a-f]{16}$`)
	proofReasonPattern  = regexp.MustCompile(`^(run[0-9]{1,2}:)?[a-z_]{1,64}$`)
	proofOutcomePattern = regexp.MustCompile(`^(ok|ok_unparsed|http_[0-9]{3}|redirect_rejected|oversize|transport_error|transport_panic|body_read_error)$`)
	proofDigestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func proofValidTime(s string, allowEmpty bool) bool {
	if s == "" {
		return allowEmpty
	}
	_, err := time.Parse(time.RFC3339Nano, s)
	return err == nil
}

func proofValidDisposition(d ProofDisposition, allowNotRun bool) bool {
	return d == ProofPass || d == ProofFail || d == ProofIncomplete || (allowNotRun && d == proofNotRun)
}

func proofValidReasons(rs []string) bool {
	for _, r := range rs {
		if !proofReasonPattern.MatchString(r) {
			return false
		}
	}
	return true
}

func proofValidDiff(d ProofSetDiff) bool {
	if d.Expected < 0 || d.Observed < 0 {
		return false
	}
	for _, list := range [][]string{d.Missing, d.Extra, d.AdapterOmitted} {
		for _, ref := range list {
			if !proofRefPattern.MatchString(ref) {
				return false
			}
		}
	}
	for _, m := range d.Mismapped {
		if !proofRefPattern.MatchString(m.Ref) || len(m.Fields) == 0 {
			return false
		}
		for _, f := range m.Fields {
			switch f {
			case "last_message_at", "sent_at", "sender_type", "content_type", "attachments":
			default:
				return false
			}
		}
	}
	return true
}

func validProofReceipt(r *ProofReceipt) bool {
	if r == nil || r.SchemaVersion != ProofSchemaVersion || r.EvidenceType != ProofEvidenceType || r.Live || r.Governance ||
		!proofSHAPattern.MatchString(r.SourceSHA) || !proofValidTime(r.StartedAt, false) || !proofValidTime(r.FinishedAt, false) ||
		!proofValidDisposition(r.Disposition, false) || !proofValidReasons(r.Reasons) {
		return false
	}
	iv, op := r.Inventory, r.Options
	if !proofDigestPattern.MatchString(iv.Digest) || !proofLabelPattern.MatchString(iv.Provenance) || !proofRefPattern.MatchString(iv.PageRef) ||
		!proofValidTime(iv.Since, false) || iv.ConversationCount < 0 || iv.MessageCount < 0 ||
		op.MaxAttempts < 1 || op.MaxAttempts > ProofMaxAttempts || op.MaxDurationMS < 1 || op.ResponseCeiling != ProofMaxResponseBytes || op.Runs != proofRuns ||
		r.Budget.AttemptsUsed < 0 || r.Budget.Denied < 0 || len(r.Runs) != proofRuns {
		return false
	}
	for _, d := range r.Denials {
		if !proofReasonPattern.MatchString(d.Reason) {
			return false
		}
	}
	for i, run := range r.Runs {
		if run.Index != i+1 || !proofValidDisposition(run.Disposition, true) || !proofValidReasons(run.Reasons) ||
			!proofValidTime(run.StartedAt, true) || !proofValidTime(run.FinishedAt, true) ||
			!proofValidDiff(run.Conversations) || !proofValidDiff(run.Messages) {
			return false
		}
		for _, q := range run.Requests {
			if (q.Kind != "conversations" && q.Kind != "messages") || (q.Template != proofConvTemplate && q.Template != proofMsgTemplate) ||
				!proofOutcomePattern.MatchString(q.Outcome) || q.Seq < 1 || q.Rows < 0 || q.ResponseBytes < 0 {
				return false
			}
			for _, ref := range []string{q.ConversationRef, q.CursorRef} {
				if ref != "" && !proofRefPattern.MatchString(ref) {
					return false
				}
			}
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Pseudonyms and digest

func proofPseudonym(key []byte, domain, value string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte("cvf-pancake-proof/v1\x00" + domain + "\x00" + value))
	return domain[:3] + "_" + hex.EncodeToString(m.Sum(nil))[:16]
}

func proofInstant(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func proofInventoryDigest(inv *ProofInventory) string {
	type att struct{ T, N string }
	type msg struct {
		ID, At, Sender, CT string
		Atts               []att
	}
	type conv struct {
		ID, At string
		Msgs   []msg
	}
	out := struct {
		Prov, Page string
		Convs      []conv
	}{Prov: inv.Provenance, Page: inv.PageID}
	for _, c := range inv.Conversations {
		cc := conv{ID: c.ID, At: proofInstant(c.UpdatedAt)}
		for _, m := range c.Messages {
			mm := msg{ID: m.ID, At: proofInstant(m.SentAt), Sender: m.SenderType, CT: m.ContentType}
			for _, a := range m.Attachments {
				mm.Atts = append(mm.Atts, att{a.Type, a.Name})
			}
			cc.Msgs = append(cc.Msgs, mm)
		}
		sort.Slice(cc.Msgs, func(i, j int) bool { return cc.Msgs[i].ID < cc.Msgs[j].ID })
		out.Convs = append(out.Convs, cc)
	}
	sort.Slice(out.Convs, func(i, j int) bool { return out.Convs[i].ID < out.Convs[j].ID })
	b, _ := json.Marshal(out)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ---------------------------------------------------------------------------
// Row observer: independent view of physical rows from the transport

type proofObserver struct {
	phys, dup, old, nonInbox int
	seen                     map[string]bool
	eligible                 map[string]bool
	order                    []string
	lastRows                 int
	lastOK                   bool
}

func newProofObserver() *proofObserver {
	return &proofObserver{seen: map[string]bool{}, eligible: map[string]bool{}}
}

// observe parses one 2xx body. field is "conversations" or "messages".
func (o *proofObserver) observe(body []byte, field string, since time.Time, convPage bool) (rows int, parsed bool, ids []string) {
	var env map[string]json.RawMessage
	if json.Unmarshal(body, &env) != nil {
		o.lastRows, o.lastOK = 0, false
		return 0, false, nil
	}
	var list []map[string]interface{}
	raw := bytes.TrimSpace(env[field])
	if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &list) != nil {
		o.lastRows, o.lastOK = 0, false
		return 0, false, nil
	}
	for _, row := range list {
		id, _ := row["id"].(string)
		stamp := ""
		if convPage {
			stamp, _ = row["updated_at"].(string)
		} else {
			stamp, _ = row["inserted_at"].(string)
		}
		o.phys++
		ids = append(ids, id)
		if convPage {
			if typ, _ := row["type"].(string); typ != "" && typ != "INBOX" {
				o.nonInbox++
				continue
			}
		}
		at := parsePancakeTime(stamp)
		if id == "" || at.IsZero() {
			continue
		}
		if convPage {
			// Conversation contract: the since filter runs before dedupe, so an old occurrence
			// must not hide a later eligible row with the same id.
			if at.Before(since) {
				o.old++
				continue
			}
			if o.seen[id] {
				o.dup++
				continue
			}
			o.seen[id] = true
		} else {
			// Message contract: dedupe first, then the since filter.
			if o.seen[id] {
				o.dup++
				continue
			}
			o.seen[id] = true
			if at.Before(since) {
				o.old++
				continue
			}
		}
		o.eligible[id] = true
		o.order = append(o.order, id)
	}
	o.lastRows, o.lastOK = len(list), true
	return len(list), true, ids
}

func (o *proofObserver) counts(mapped int) ProofRowCounts {
	return ProofRowCounts{PhysicalRows: o.phys, DuplicateRow: o.dup, OldRows: o.old, NonInboxRows: o.nonInbox, RawEligible: len(o.eligible), Mapped: mapped}
}

func (o *proofObserver) terminal() bool { return o.lastOK && o.lastRows == 0 }

// ---------------------------------------------------------------------------
// Guarded transport: admission, shared budget, redirect/size rejection, observation

type proofTransport struct {
	inner    http.RoundTripper
	opts     *PancakeProofOptions
	approved map[string]bool
	now      func() time.Time
	start    time.Time

	mu       sync.Mutex
	attempts int
	denied   []ProofDenial
	records  []ProofRequestRecord
	run      int

	// flags describing what stopped a fetch (read by classifyProofError)
	budgetHit, deadlineHit, deniedHit, redirectHit, oversizeHit, panicHit, innerHit, readHit bool

	// per-run observation state
	convObs  *proofObserver
	msgObs   map[string]*proofObserver
	msgTerm  map[string]bool
	curConv  string // conversation currently being fetched (messages)
	convSeen map[string]bool
	untils   []string
}

func (t *proofTransport) beginRun(run int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.run = run
	t.convObs = newProofObserver()
	t.msgObs = map[string]*proofObserver{}
	t.msgTerm = map[string]bool{}
	t.convSeen = map[string]bool{}
	t.untils = nil
	t.curConv = ""
	t.budgetHit, t.deadlineHit, t.deniedHit, t.redirectHit, t.oversizeHit, t.panicHit, t.innerHit, t.readHit = false, false, false, false, false, false, false, false
}

func (t *proofTransport) setConv(id string) {
	t.mu.Lock()
	t.curConv = id
	if id != "" {
		t.msgObs[id] = newProofObserver()
	}
	t.mu.Unlock()
}

func (t *proofTransport) ref(domain, v string) string {
	return proofPseudonym(t.opts.PseudonymKey, domain, v)
}

// admit checks a request against the exact allowlist. It returns the kind, conversation id and
// a denial reason ("" when admitted). It must not touch the network.
func (t *proofTransport) admit(req *http.Request) (kind, convID, reason string) {
	if req == nil || req.URL == nil {
		return "", "", "nil_request"
	}
	u := req.URL
	switch {
	case req.Method != http.MethodGet:
		return "", "", "method"
	case u.Scheme != "https":
		return "", "", "scheme"
	case u.Host != proofHost || (req.Host != "" && req.Host != proofHost):
		return "", "", "host"
	case u.User != nil:
		return "", "", "userinfo"
	case u.Fragment != "" || u.RawFragment != "" || u.Opaque != "":
		return "", "", "fragment_or_opaque"
	}
	if !proofSafeSegment(t.opts.PageID) {
		return "", "", "unsafe_page"
	}
	page := url.PathEscape(t.opts.PageID)
	path := u.EscapedPath()
	convPath := "/api/public_api/v2/pages/" + page + "/conversations"
	msgPrefix := "/api/public_api/v1/pages/" + page + "/conversations/"
	switch {
	case path == convPath:
		kind = "conversations"
	case strings.HasPrefix(path, msgPrefix) && strings.HasSuffix(path, "/messages"):
		esc := strings.TrimSuffix(strings.TrimPrefix(path, msgPrefix), "/messages")
		id, err := url.PathUnescape(esc)
		if err != nil || !proofSafeSegment(id) || url.PathEscape(id) != esc || !t.approved[id] {
			return "", "", "path"
		}
		kind, convID = "messages", id
	default:
		return "", "", "path"
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || strings.Contains(u.RawQuery, ";") {
		return "", "", "query"
	}
	for _, v := range q {
		if len(v) != 1 {
			return "", "", "query_duplicate"
		}
	}
	if tok := q.Get("page_access_token"); subtle.ConstantTimeCompare([]byte(tok), []byte(t.opts.Token)) != 1 {
		return "", "", "query_token"
	}
	delete(q, "page_access_token")
	if kind == "messages" {
		for k, v := range q {
			if k != "current_count" || !proofDigits.MatchString(v[0]) {
				return "", "", "query_key"
			}
		}
		return kind, convID, ""
	}
	for k, v := range q {
		switch k {
		case "type":
			if v[0] != "INBOX" {
				return "", "", "query_value"
			}
		case "order_by":
			if v[0] != "updated_at" {
				return "", "", "query_value"
			}
		case "until":
			if !proofDigits.MatchString(v[0]) {
				return "", "", "query_value"
			}
		case "since":
			if v[0] != strconv.FormatInt(t.opts.Since.Unix(), 10) {
				return "", "", "query_value"
			}
		case "last_conversation_id":
			t.mu.Lock()
			ok := t.convSeen[v[0]]
			t.mu.Unlock()
			if !ok {
				return "", "", "cursor_unobserved"
			}
		default:
			return "", "", "query_key"
		}
	}
	if q.Get("type") == "" || q.Get("order_by") == "" || q.Get("until") == "" {
		return "", "", "query_missing"
	}
	return kind, "", ""
}

func (t *proofTransport) deny(reason string) error {
	t.mu.Lock()
	t.denied = append(t.denied, ProofDenial{Run: t.run, Reason: reason})
	t.deniedHit = true
	t.mu.Unlock()
	return errProofDenied
}

func (t *proofTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	kind, convID, reason := t.admit(req)
	if reason != "" {
		return nil, t.deny(reason)
	}
	// Shared budget: checked before the attempt is allowed to reach the transport.
	t.mu.Lock()
	if t.attempts >= t.opts.MaxAttempts {
		t.budgetHit = true
		t.mu.Unlock()
		return nil, errProofBudget
	}
	if ctxErr := req.Context().Err(); ctxErr != nil || t.now().Sub(t.start) >= t.opts.MaxDuration {
		t.deadlineHit = true
		t.mu.Unlock()
		return nil, errProofBudget
	}
	t.attempts++
	rec := ProofRequestRecord{Seq: t.attempts, Run: t.run, Kind: kind}
	t.mu.Unlock()

	q := req.URL.Query()
	if kind == "conversations" {
		rec.Template = proofConvTemplate
		if c := q.Get("last_conversation_id"); c != "" {
			rec.CursorRef = t.ref("cursor", c)
		}
		t.mu.Lock()
		t.untils = append(t.untils, q.Get("until"))
		t.mu.Unlock()
	} else {
		rec.Template = proofMsgTemplate
		rec.ConversationRef = t.ref("conversation", convID)
		rec.CurrentCount, _ = strconv.Atoi(q.Get("current_count"))
	}

	finish := func(outcome string) {
		rec.Outcome = outcome
		t.mu.Lock()
		t.records = append(t.records, rec)
		t.mu.Unlock()
	}
	defer func() {
		if p := recover(); p != nil { // inner panic text is never recorded
			t.mu.Lock()
			t.panicHit = true
			t.mu.Unlock()
			resp, err = nil, errProofInner
			finish("transport_panic")
		}
	}()

	res, ierr := t.inner.RoundTrip(req)
	if ierr != nil || res == nil {
		t.mu.Lock()
		t.innerHit = true
		t.mu.Unlock()
		finish("transport_error")
		return nil, errProofInner
	}
	rec.Status = res.StatusCode
	if res.StatusCode >= 300 && res.StatusCode < 400 {
		if res.Body != nil {
			res.Body.Close()
		}
		t.mu.Lock()
		t.redirectHit = true
		t.mu.Unlock()
		finish("redirect_rejected")
		return nil, errProofRedirect
	}
	var body []byte
	if res.Body != nil {
		var rerr error
		body, rerr = io.ReadAll(io.LimitReader(res.Body, ProofMaxResponseBytes+1))
		res.Body.Close()
		if rerr != nil { // a failed read can never establish content or an explicit-empty terminal
			rec.ResponseBytes = len(body)
			t.mu.Lock()
			t.readHit = true
			t.mu.Unlock()
			finish("body_read_error")
			return nil, errProofRead
		}
	}
	rec.ResponseBytes = len(body)
	if len(body) > ProofMaxResponseBytes {
		t.mu.Lock()
		t.oversizeHit = true
		t.mu.Unlock()
		finish("oversize")
		return nil, errProofOversize
	}
	outcome := "http_" + strconv.Itoa(res.StatusCode)
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		t.mu.Lock()
		var obs *proofObserver
		field, convPage := "messages", false
		if kind == "conversations" {
			obs, field, convPage = t.convObs, "conversations", true
		} else {
			obs = t.msgObs[convID]
		}
		if obs != nil {
			rows, ok, ids := obs.observe(body, field, t.opts.Since, convPage)
			rec.Rows, rec.EmptyTerminal = rows, ok && rows == 0
			if convPage {
				for _, id := range ids {
					t.convSeen[id] = true
				}
			}
			outcome = "ok"
			if !ok {
				outcome = "ok_unparsed"
			}
		}
		t.mu.Unlock()
	}
	finish(outcome)
	return &http.Response{
		Status: res.Status, StatusCode: res.StatusCode, Header: http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: req, Proto: "HTTP/1.1", ProtoMajor: 1, ProtoMinor: 1,
	}, nil
}

// ---------------------------------------------------------------------------
// Proof execution

func newProofAdapter(o *PancakeProofOptions, rt http.RoundTripper) *PancakeAdapter {
	return &PancakeAdapter{
		creds: PancakeCredentials{PageID: o.PageID, PageAccessToken: o.Token},
		client: &http.Client{
			Transport:     rt,
			Timeout:       30 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		v1Base:      pancakePublicV1,
		v2Base:      pancakePublicV2,
		minInterval: o.MinRequestInterval,
		backoff:     o.RetryBackoff,
	}
}

func classifyProofError(err error, ctx context.Context, t *proofTransport) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch {
	case t.panicHit:
		return "transport_panic"
	case t.readHit:
		return "response_read_error"
	case t.deniedHit:
		return "request_denied"
	case t.redirectHit:
		return "redirect_rejected"
	case t.oversizeHit:
		return "response_oversize"
	case t.budgetHit:
		return "budget_exhausted"
	case t.deadlineHit, ctx.Err() != nil:
		return "deadline_or_cancelled"
	case t.innerHit:
		return "transport_error"
	case errors.Is(err, ErrPancakeCoverageIncomplete):
		return "conversation_coverage_incomplete"
	case errors.Is(err, ErrPancakeMessageCoverageIncomplete):
		return "message_coverage_incomplete"
	}
	return "adapter_error"
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func attachmentsEqual(exp []ProofAttachment, got []Attachment) bool {
	if len(exp) != len(got) {
		return false
	}
	for i := range exp {
		if exp[i].Type != got[i].Type || exp[i].Name != got[i].Name {
			return false
		}
	}
	return true
}

func newDiff(expected, observed int) ProofSetDiff {
	return ProofSetDiff{Expected: expected, Observed: observed, Missing: []string{}, Extra: []string{}, Mismapped: []ProofMismatch{}, AdapterOmitted: []string{}}
}

func (d *ProofSetDiff) failing() bool {
	return len(d.Missing)+len(d.Extra)+len(d.Mismapped)+len(d.AdapterOmitted) > 0
}

// RunPancakeProof executes two guarded traversals against inv and returns a sanitized receipt.
// Setup problems return ErrPancakeProofInput before any adapter work.
func RunPancakeProof(ctx context.Context, opts PancakeProofOptions, inv *ProofInventory) (receipt *ProofReceipt, err error) {
	if err := opts.validate(inv); err != nil {
		return nil, err
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	start := opts.Now()
	runCtx, cancel := context.WithTimeout(ctx, opts.MaxDuration)
	defer cancel()

	tr := &proofTransport{inner: opts.Transport, opts: &opts, approved: map[string]bool{}, now: opts.Now, start: start}
	totalMsgs := 0
	for _, c := range inv.Conversations {
		tr.approved[c.ID] = true
		totalMsgs += len(c.Messages)
	}
	adapter := newProofAdapter(&opts, tr)

	rc := &ProofReceipt{
		SchemaVersion: ProofSchemaVersion, EvidenceType: ProofEvidenceType, SourceSHA: opts.SourceSHA,
		StartedAt: proofInstant(start),
		Inventory: ProofInvSummary{
			Digest: proofInventoryDigest(inv), Provenance: inv.Provenance, PageRef: proofPseudonym(opts.PseudonymKey, "page", opts.PageID),
			Since: proofInstant(opts.Since), ConversationCount: len(inv.Conversations), MessageCount: totalMsgs,
		},
		Options: ProofOptSummary{
			MaxAttempts: opts.MaxAttempts, MaxDurationMS: opts.MaxDuration.Milliseconds(), MinRequestInterval: opts.MinRequestInterval.Milliseconds(),
			RetryBackoffMS: opts.RetryBackoff.Milliseconds(), ResponseCeiling: ProofMaxResponseBytes, Runs: proofRuns,
		},
		Denials: []ProofDenial{}, Runs: []ProofRunReport{}, Reasons: []string{},
	}

	skipRest := false
	for i := 1; i <= proofRuns; i++ {
		if skipRest {
			rc.Runs = append(rc.Runs, ProofRunReport{Index: i, Requests: []ProofRequestRecord{}, Disposition: proofNotRun, Reasons: []string{"previous_run_incomplete"},
				Conversations: newDiff(0, 0), Messages: newDiff(0, 0)})
			continue
		}
		rep := runProofOnce(runCtx, i, adapter, tr, &opts, inv, totalMsgs)
		if rep.Disposition == ProofIncomplete {
			skipRest = true
		}
		rc.Runs = append(rc.Runs, rep)
	}

	end := opts.Now()
	rc.FinishedAt = proofInstant(end)
	tr.mu.Lock()
	rc.Budget = ProofBudget{AttemptsUsed: tr.attempts, Denied: len(tr.denied), ElapsedMS: end.Sub(start).Milliseconds()}
	rc.Denials = append(rc.Denials, tr.denied...)
	tr.mu.Unlock()

	rc.Disposition = ProofPass
	for _, r := range rc.Runs {
		switch {
		case r.Disposition == ProofFail:
			rc.Disposition = ProofFail
		case r.Disposition != ProofPass && rc.Disposition != ProofFail:
			rc.Disposition = ProofIncomplete
		}
		for _, reason := range r.Reasons {
			rc.Reasons = append(rc.Reasons, fmt.Sprintf("run%d:%s", r.Index, reason))
		}
	}
	return rc, nil
}

func runProofOnce(ctx context.Context, idx int, adapter *PancakeAdapter, tr *proofTransport, opts *PancakeProofOptions, inv *ProofInventory, totalMsgs int) (rep ProofRunReport) {
	tr.beginRun(idx)
	rep = ProofRunReport{Index: idx, StartedAt: proofInstant(opts.Now()), UntilStable: true,
		Conversations: newDiff(len(inv.Conversations), 0), Messages: newDiff(totalMsgs, 0), Requests: []ProofRequestRecord{}, Reasons: []string{}}
	incomplete := func(reason string) { rep.Reasons = append(rep.Reasons, reason); rep.Disposition = ProofIncomplete }
	// captureUntil finalizes the observed conversation `until` values; it is idempotent and is
	// evaluated before any disposition so UntilStable=false can never coexist with PASS.
	captureUntil := func() {
		tr.mu.Lock()
		defer tr.mu.Unlock()
		rep.UntilStable = true
		if len(tr.untils) > 0 {
			u, _ := strconv.ParseInt(tr.untils[0], 10, 64)
			rep.ConversationUntil = u
			for _, s := range tr.untils {
				if s != tr.untils[0] {
					rep.UntilStable = false
				}
			}
		}
	}
	defer func() {
		if p := recover(); p != nil { // panic text is never recorded
			rep.Reasons = append(rep.Reasons, "internal_panic")
			rep.Disposition = ProofIncomplete
		}
		tr.mu.Lock()
		for _, r := range tr.records {
			if r.Run == idx {
				rep.Requests = append(rep.Requests, r)
			}
		}
		tr.mu.Unlock()
		captureUntil()
		sort.SliceStable(rep.Requests, func(i, j int) bool { return rep.Requests[i].Seq < rep.Requests[j].Seq })
		rep.FinishedAt = proofInstant(opts.Now())
		if rep.Disposition == "" {
			rep.Disposition = ProofPass
		}
	}()

	convs, err := adapter.FetchRecentConversations(ctx, opts.Since, 0)
	if err != nil {
		incomplete(classifyProofError(err, ctx, tr))
		return rep
	}
	tr.mu.Lock()
	cobs := tr.convObs
	tr.mu.Unlock()
	rep.ConversationTerm = cobs.terminal()
	rep.ConversationRows = cobs.counts(len(convs))
	if !rep.ConversationTerm {
		incomplete("conversation_terminal_missing")
		return rep
	}

	expected := map[string]ProofConversation{}
	for _, c := range inv.Conversations {
		expected[c.ID] = c
	}
	mapped := map[string]SyncedConversation{}
	for _, c := range convs {
		mapped[c.ExternalID] = c
	}
	cd := newDiff(len(expected), len(mapped))
	for _, id := range sortedKeys(boolSet(expected)) {
		m, ok := mapped[id]
		if !ok {
			cd.Missing = append(cd.Missing, tr.ref("conversation", id))
			continue
		}
		if !m.LastMessageAt.Equal(expected[id].UpdatedAt) {
			cd.Mismapped = append(cd.Mismapped, ProofMismatch{Ref: tr.ref("conversation", id), Fields: []string{"last_message_at"}})
		}
	}
	for id := range mapped {
		if _, ok := expected[id]; !ok {
			cd.Extra = append(cd.Extra, tr.ref("conversation", id))
		}
	}
	for id := range cobs.eligible { // physical row eligible but dropped by adapter mapping
		if _, ok := mapped[id]; !ok {
			cd.AdapterOmitted = append(cd.AdapterOmitted, tr.ref("conversation", id))
		}
	}
	sort.Strings(cd.Extra)
	sort.Strings(cd.AdapterOmitted)
	rep.Conversations = cd

	// Messages only for conversations that are both observed and approved by the inventory.
	var fetchIDs []string
	for id := range mapped {
		if _, ok := expected[id]; ok {
			fetchIDs = append(fetchIDs, id)
		}
	}
	sort.Strings(fetchIDs)
	md := newDiff(totalMsgs, 0)
	var msgPhys, msgDup, msgOld, msgElig, msgMapped int
	for _, id := range fetchIDs {
		tr.setConv(id)
		msgs, merr := adapter.FetchMessages(ctx, id, opts.Since)
		rep.MessageFetches++
		if merr != nil {
			incomplete(classifyProofError(merr, ctx, tr))
			return rep
		}
		tr.mu.Lock()
		mo := tr.msgObs[id]
		tr.mu.Unlock()
		if !mo.terminal() {
			incomplete("message_terminal_missing")
			return rep
		}
		rep.MessageTerminals++
		msgPhys += mo.phys
		msgDup += mo.dup
		msgOld += mo.old
		msgElig += len(mo.eligible)
		msgMapped += len(msgs)
		md.Observed += len(msgs)
		expMsgs := map[string]ProofMessage{}
		for _, m := range expected[id].Messages {
			expMsgs[m.ID] = m
		}
		got := map[string]SyncedMessage{}
		for _, m := range msgs {
			got[m.ExternalID] = m
		}
		for _, mid := range sortedKeys(boolSetMsg(expMsgs)) {
			g, ok := got[mid]
			ref := tr.ref("message", id+"\x00"+mid)
			if !ok {
				md.Missing = append(md.Missing, ref)
				continue
			}
			e := expMsgs[mid]
			var bad []string
			if !g.SentAt.Equal(e.SentAt) {
				bad = append(bad, "sent_at")
			}
			if g.SenderType != e.SenderType {
				bad = append(bad, "sender_type")
			}
			if g.ContentType != e.ContentType {
				bad = append(bad, "content_type")
			}
			if !attachmentsEqual(e.Attachments, g.Attachments) {
				bad = append(bad, "attachments")
			}
			if len(bad) > 0 {
				md.Mismapped = append(md.Mismapped, ProofMismatch{Ref: ref, Fields: bad})
			}
		}
		for mid := range got {
			if _, ok := expMsgs[mid]; !ok {
				md.Extra = append(md.Extra, tr.ref("message", id+"\x00"+mid))
			}
		}
		for mid := range mo.eligible {
			if _, ok := got[mid]; !ok {
				md.AdapterOmitted = append(md.AdapterOmitted, tr.ref("message", id+"\x00"+mid))
			}
		}
	}
	sort.Strings(md.Extra)
	sort.Strings(md.AdapterOmitted)
	rep.Messages = md
	rep.MessageRows = ProofRowCounts{PhysicalRows: msgPhys, DuplicateRow: msgDup, OldRows: msgOld, RawEligible: msgElig, Mapped: msgMapped}

	switch {
	case rep.Conversations.failing() || rep.Messages.failing():
		rep.Disposition = ProofFail
		rep.Reasons = append(rep.Reasons, "reconciliation_mismatch")
	}
	captureUntil()
	if !rep.UntilStable {
		rep.Disposition = ProofFail
		rep.Reasons = append(rep.Reasons, "until_unstable")
	}
	if rep.Disposition != ProofFail {
		if len(expected) == 0 {
			incomplete("empty_inventory")
		} else if totalMsgs == 0 {
			incomplete("empty_message_inventory")
		}
	}
	return rep
}

func boolSet(m map[string]ProofConversation) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

func boolSetMsg(m map[string]ProofMessage) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

// ProofSensitiveValues lists raw values that must never appear in a receipt for opts/inv.
func ProofSensitiveValues(opts PancakeProofOptions, inv *ProofInventory) []string {
	out := []string{opts.Token, opts.PageID, string(opts.PseudonymKey)}
	if inv != nil {
		for _, c := range inv.Conversations {
			out = append(out, c.ID)
			for _, m := range c.Messages {
				out = append(out, m.ID)
				for _, a := range m.Attachments {
					out = append(out, a.Name)
				}
			}
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Finite in-memory transcript transport (the only transport R034 tooling supplies)

// ProofResponse is one scripted reply. Err simulates a transport failure (its text is never surfaced).
type ProofResponse struct {
	Status   int
	Body     string
	Location string // optional redirect target (3xx)
	Err      error
	Block    bool // block until the request context ends (deadline/cancel tests)
}

// ProofTranscript maps a route key to ordered replies. Keys: "conv:<cursor>" ("conv:" for the first
// page) and "msg:<conversationID>:<current_count>" (0 for the first page). A key's last reply is
// reused when Repeat is set; otherwise exhaustion is an error. Total calls are hard-capped.
type ProofTranscript struct {
	Routes  map[string][]ProofResponse
	Repeat  bool
	Ceiling int

	mu    sync.Mutex
	used  map[string]int
	calls int
}

// Calls reports how many requests reached the transcript (the underlying transport).
func (p *ProofTranscript) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func (p *ProofTranscript) RoundTrip(req *http.Request) (*http.Response, error) {
	p.mu.Lock()
	p.calls++
	ceil := p.Ceiling
	if ceil <= 0 {
		ceil = proofTranscriptCeil
	}
	if p.calls > ceil {
		p.mu.Unlock()
		return nil, errors.New("transcript ceiling")
	}
	if p.used == nil {
		p.used = map[string]int{}
	}
	key := ""
	q := req.URL.Query()
	if strings.HasSuffix(req.URL.Path, "/messages") {
		parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
		cc := q.Get("current_count")
		if cc == "" {
			cc = "0"
		}
		key = "msg:" + parts[len(parts)-2] + ":" + cc
	} else {
		key = "conv:" + q.Get("last_conversation_id")
	}
	list := p.Routes[key]
	n := p.used[key]
	p.used[key] = n + 1
	var r ProofResponse
	switch {
	case n < len(list):
		r = list[n]
	case p.Repeat && len(list) > 0:
		r = list[len(list)-1]
	default:
		p.mu.Unlock()
		return nil, errors.New("unscripted request")
	}
	p.mu.Unlock()
	if r.Block { // finite: a mutated deadline guard cannot hang the process
		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(proofBlockSafety):
			return nil, errors.New("transcript block safety expired")
		}
	}
	if r.Err != nil {
		return nil, r.Err
	}
	h := http.Header{}
	if r.Location != "" {
		h.Set("Location", r.Location)
	}
	return &http.Response{StatusCode: r.Status, Status: strconv.Itoa(r.Status), Header: h, Body: io.NopCloser(strings.NewReader(r.Body)), Request: req}, nil
}

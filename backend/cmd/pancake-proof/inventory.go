package main

// Explicit synthetic expected-inventory input for the offline proof CLI (CCMAI-RUNTIME-041).
//
// The file is caller-authored synthetic test input. It only replaces the expected inventory;
// the HTTP transcript, page, token, since and transport remain the fixed synthetic constants.
// Nothing here discovers files, reads the environment, opens a network connection or echoes
// input back: every failure carries an internal code for tests and a fixed message for users.

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/CVF-Ecosystem/Customer-Care-Monitor/backend/channels"
)

const (
	inventorySchema = "pancake-proof-synthetic-inventory/1"

	maxInventoryBytes       = 1 << 20
	maxInventoryConvs       = 100
	maxInventoryMessages    = 1000
	maxInventoryAttachments = 16
	maxInventoryDepth       = 64
	maxInventoryIDBytes     = 128
	maxInventoryNameBytes   = 256
)

// inventoryError is the only error the loader returns. Code is for tests; Error() is fixed.
type inventoryError struct{ code string }

func (e *inventoryError) Error() string { return "inventory rejected" }

func reject(code string) error { return &inventoryError{code: code} }

func inventoryCode(err error) string {
	var ie *inventoryError
	if errors.As(err, &ie) {
		return ie.code
	}
	return "other"
}

var inventoryLabelPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

// safeSegment mirrors the proof library's identifier rule (no dot segments, separators, escapes,
// query/fragment markers or control characters), plus no replacement characters.
func safeSegment(v string) bool {
	if v == "" || v == "." || v == ".." {
		return false
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f || r == utf8.RuneError || r == ' ' || strings.ContainsRune(`/\%?#;:`, r) {
			return false
		}
	}
	return true
}

// loadInventoryFile admits one explicit local regular file and parses it with a bounded read.
func loadInventoryFile(path string) (*channels.ProofInventory, error) {
	if path == "" || path == "-" || strings.ContainsRune(path, 0) || strings.Contains(path, "://") ||
		strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, "//") {
		return nil, reject("path")
	}
	before, err := os.Lstat(path) // Lstat: a symlink or reparse point is not a regular file
	if err != nil || !before.Mode().IsRegular() {
		return nil, reject("not_regular_file")
	}
	if before.Size() > maxInventoryBytes {
		return nil, reject("too_large")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, reject("open")
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, reject("not_regular_file")
	}
	return parseInventory(f)
}

// parseInventory reads at most maxInventoryBytes+1 bytes, never accepting a truncated prefix.
func parseInventory(r io.Reader) (*channels.ProofInventory, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxInventoryBytes+1))
	if err != nil {
		return nil, reject("read")
	}
	if len(data) > maxInventoryBytes {
		return nil, reject("too_large")
	}
	if !utf8.Valid(data) {
		return nil, reject("utf8")
	}
	tree, err := decodeStrict(data)
	if err != nil {
		return nil, err
	}
	return convertInventory(tree)
}

// decodeStrict parses one JSON value into a generic tree, rejecting duplicate keys, excess
// nesting and trailing data. encoding/json is used only as a tokenizer.
func decodeStrict(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec, 0)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, reject("trailing_data")
	}
	return v, nil
}

func decodeValue(dec *json.Decoder, depth int) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, reject("syntax")
	}
	d, isDelim := tok.(json.Delim)
	if !isDelim {
		return tok, nil // string, json.Number, bool or nil
	}
	if depth+1 > maxInventoryDepth {
		return nil, reject("depth")
	}
	switch d {
	case '{':
		obj := map[string]any{}
		for dec.More() {
			kt, err := dec.Token()
			if err != nil {
				return nil, reject("syntax")
			}
			key, ok := kt.(string)
			if !ok {
				return nil, reject("syntax")
			}
			if _, dup := obj[key]; dup {
				return nil, reject("duplicate_key")
			}
			val, err := decodeValue(dec, depth+1)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
		if _, err := dec.Token(); err != nil {
			return nil, reject("syntax")
		}
		return obj, nil
	case '[':
		arr := []any{}
		for dec.More() {
			val, err := decodeValue(dec, depth+1)
			if err != nil {
				return nil, err
			}
			arr = append(arr, val)
		}
		if _, err := dec.Token(); err != nil {
			return nil, reject("syntax")
		}
		return arr, nil
	}
	return nil, reject("syntax")
}

// Explicit wire structs: the file format is defined here, not by the proof library's Go types.
type wireInventory struct {
	schema, evidence, provenance, pageID string
	conversations                        []any
}

// exactObject requires the object to contain exactly the wanted keys (each appears once by
// construction): missing and unknown fields are both rejected. Keys match case-sensitively.
func exactObject(v any, keys ...string) (map[string]any, error) {
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, reject("type")
	}
	for _, k := range keys {
		if _, present := obj[k]; !present {
			return nil, reject("missing_field")
		}
	}
	if len(obj) != len(keys) {
		return nil, reject("unknown_field")
	}
	return obj, nil
}

func str(obj map[string]any, key string) (string, error) {
	s, ok := obj[key].(string)
	if !ok {
		return "", reject("type")
	}
	return s, nil
}

func array(obj map[string]any, key string) ([]any, error) {
	a, ok := obj[key].([]any)
	if !ok {
		return nil, reject("type")
	}
	return a, nil
}

func instant(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, s) // requires an explicit Z or numeric offset
	if err != nil || t.IsZero() {
		return time.Time{}, reject("time")
	}
	return t.UTC(), nil
}

func identifier(s string) bool {
	return strings.HasPrefix(s, "syn-") && len(s) <= maxInventoryIDBytes && safeSegment(s)
}

func convertInventory(tree any) (*channels.ProofInventory, error) {
	root, err := exactObject(tree, "schema_version", "evidence_type", "provenance", "page_id", "conversations")
	if err != nil {
		return nil, err
	}
	var w wireInventory
	for _, f := range []struct {
		key string
		dst *string
	}{{"schema_version", &w.schema}, {"evidence_type", &w.evidence}, {"provenance", &w.provenance}, {"page_id", &w.pageID}} {
		if *f.dst, err = str(root, f.key); err != nil {
			return nil, err
		}
	}
	if w.conversations, err = array(root, "conversations"); err != nil {
		return nil, err
	}
	switch {
	case w.schema != inventorySchema:
		return nil, reject("schema_version")
	case w.evidence != "SYNTHETIC_OFFLINE":
		return nil, reject("evidence_type")
	case !inventoryLabelPattern.MatchString(w.provenance) || !strings.HasPrefix(w.provenance, "synthetic-"):
		return nil, reject("provenance")
	case w.pageID != fixturePage:
		return nil, reject("page_id")
	case len(w.conversations) > maxInventoryConvs:
		return nil, reject("too_many_conversations")
	}

	inv := &channels.ProofInventory{Provenance: w.provenance, PageID: w.pageID}
	seenConv := map[string]bool{}
	totalMsgs := 0
	for _, cv := range w.conversations {
		co, err := exactObject(cv, "id", "updated_at", "messages")
		if err != nil {
			return nil, err
		}
		id, err := str(co, "id")
		if err != nil {
			return nil, err
		}
		updated, err := str(co, "updated_at")
		if err != nil {
			return nil, err
		}
		msgs, err := array(co, "messages")
		if err != nil {
			return nil, err
		}
		if !identifier(id) || seenConv[id] {
			return nil, reject("conversation_id")
		}
		seenConv[id] = true
		ut, err := instant(updated)
		if err != nil {
			return nil, err
		}
		if totalMsgs += len(msgs); totalMsgs > maxInventoryMessages {
			return nil, reject("too_many_messages")
		}
		conv := channels.ProofConversation{ID: id, UpdatedAt: ut}
		seenMsg := map[string]bool{}
		for _, mv := range msgs {
			mo, err := exactObject(mv, "id", "sent_at", "sender_type", "content_type", "attachments")
			if err != nil {
				return nil, err
			}
			mid, err := str(mo, "id")
			if err != nil {
				return nil, err
			}
			sent, err := str(mo, "sent_at")
			if err != nil {
				return nil, err
			}
			sender, err := str(mo, "sender_type")
			if err != nil {
				return nil, err
			}
			content, err := str(mo, "content_type")
			if err != nil {
				return nil, err
			}
			atts, err := array(mo, "attachments")
			if err != nil {
				return nil, err
			}
			if !identifier(mid) || seenMsg[mid] {
				return nil, reject("message_id")
			}
			seenMsg[mid] = true
			st, err := instant(sent)
			if err != nil {
				return nil, err
			}
			if sender != "customer" && sender != "agent" {
				return nil, reject("sender_type")
			}
			if content != "text" && content != "attachment" {
				return nil, reject("content_type")
			}
			if len(atts) > maxInventoryAttachments {
				return nil, reject("too_many_attachments")
			}
			m := channels.ProofMessage{ID: mid, SentAt: st, SenderType: sender, ContentType: content}
			for _, av := range atts {
				ao, err := exactObject(av, "type", "name")
				if err != nil {
					return nil, err
				}
				typ, err := str(ao, "type")
				if err != nil {
					return nil, err
				}
				name, err := str(ao, "name")
				if err != nil {
					return nil, err
				}
				if typ != "image" {
					return nil, reject("attachment_type")
				}
				if name == "" || len(name) > maxInventoryNameBytes || strings.ContainsFunc(name, func(r rune) bool {
					return r < 0x20 || r == 0x7f || r == utf8.RuneError
				}) {
					return nil, reject("attachment_name")
				}
				m.Attachments = append(m.Attachments, channels.ProofAttachment{Type: typ, Name: name})
			}
			conv.Messages = append(conv.Messages, m)
		}
		inv.Conversations = append(inv.Conversations, conv)
	}
	return inv, nil
}

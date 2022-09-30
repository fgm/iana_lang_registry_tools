package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	CachePath = "registry.txt"
	Url       = "https://www.iana.org/assignments/language-subtag-registry/language-subtag-registry"
)

var PropRowRx = regexp.MustCompile(`^((?:-|[[:alpha:]])+): (.+)$`)

type Date time.Time

func (d Date) IsZero() bool {
	t := time.Time(d)
	return t.IsZero()
}

func (d Date) MarshalYAML() (any, error) {
	s := time.Time(d).Format("2006-01-02")
	return s, nil
}

type Script [4]rune

// IsZero implements yaml.IsZeroer to support omitempty in yaml encoding.
func (s Script) IsZero() bool {
	var zero Script
	return s == zero
}

// MarshalYAML implements yaml.Marshaler.
func (s Script) MarshalYAML() (any, error) {
	bs := make([]byte, 4)
	for i := 0; i < 4; i++ {
		bs[i] = byte(s[i])
	}
	return string(bs), nil
}

// Entry represents a parsed block. Highest cardinalities on 30/09/2022 are:
//
//	map[string]int{
//		"Added":1,
//		"Comments":1,
//		"Deprecated":1,
//		"Description":7,
//		"File-Date":1,
//		"Macrolanguage":1,
//		"Preferred-Value":1,
//		"Prefix":11,
//		"Scope":1,
//		"Subtag":1,
//		"Suppress-Script":1,
//		"Tag":1,
//		"Type":1
//		}
type Entry struct {
	Added          Date     `yaml:"added"`                 // date only
	Comments       string   `yaml:"comments,omitempty"`    // multiline
	Deprecated     Date     `yaml:"deprecated,omitempty"`  // date only
	Description    []string `yaml:"description,omitempty"` // multiline
	MacroLanguage  string   `yaml:"macro-language,omitempty"`
	PreferredValue string   `yaml:"preferred-value,omitempty"`
	Prefix         []string `yaml:"prefix,omitempty"`          // max: 11
	Scope          string   `yaml:"scope,omitempty"`           // collection:116, macrolanguage:62, private-use:1, special:4
	Subtag         string   `yaml:"subtag,omitempty"`          // max length:10 "Qaaa..Qabx"
	SuppressScript Script   `yaml:"suppress-script,omitempty"` // length: 4
	Tag            string   `yaml:"tag,omitempty"`             // always contains a dash
	Type           string   `yaml:"type,omitempty"`            // extlang:252,grandfathered:26, language:8240, redundant:67, region:304, script:212, variant:110
}

type Registry struct {
	FileDate Date
	Entries  []Entry
}

func initRegistry(bss [][]byte) Registry {
	dateBlock := lexBlock(string(bss[0]))
	fd, ok := dateBlock["file-date"]
	if !ok {
		log.Fatalf("First block is not a file-data block: %q", dateBlock)
	}
	return Registry{FileDate: parseDate("file-date", fd)}
}

// lexlocks parses a block lexically, returning the lower-case keys and slices of values as strings.
func lexBlock(bs string) map[string][]string {
	m := make(map[string][]string, 20)
	rows := strings.Split(bs, "\n")
	var ck, cv string
	for _, row := range rows {
		if row == "" {
			continue
		}
		// New key: store the previous one
		if key := PropRowRx.FindStringSubmatch(row); len(key) > 2 {
			nk, nv := strings.ToLower(key[1]), key[2]

			if ck != "" {
				m[ck] = append(m[ck], cv)
			}
			ck, cv = nk, nv
			continue
		}
		// Not a new key: append to the current value for the current key
		cv += " " + strings.Trim(row, " ")
	}
	if ck != "" {
		m[ck] = append(m[ck], cv)
	}
	return m
}

func loadBlocks() [][]byte {
	var sep = []byte{'\n', '%', '%', '\n'}

	var (
		err     error
		f       *os.File
		res     *http.Response
		written int64
	)
	if f, err = os.Open(CachePath); err == nil {
		goto fileExists
	}
	if res, err = http.Get(Url); err != nil {
		log.Fatalf("No cache and fail to read online version: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		log.Fatalf("HTTP error getting fresh registry: %d %s\n%v", res.StatusCode, res.Status, res.Header)
	}
	if f, err = os.OpenFile(CachePath, os.O_CREATE|os.O_RDWR, 0666); err != nil {
		log.Fatalf("No cache and fail to create cache file: %v", err)
	}
	defer f.Close()
	if written, err = io.Copy(f, res.Body); err != nil {
		log.Fatalf("No cache and fail to write cache file: %v", err)
	}
	log.Printf("Written cache: %d bytes\n", written)
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		log.Fatalf("Failed resetting newly created cache file: %v", err)
	}

fileExists:
	blocks := make([][]byte, 0)
	br := bufio.NewScanner(f)
	br.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if index := bytes.Index(data, sep); index != -1 {
			advance = index + len(sep)
			token = data[:index+1]
			err = nil
			return
		}
		if !atEOF {
			return 0, nil, nil
		}
		return len(data), data, bufio.ErrFinalToken
	})
	for br.Scan() {
		block := br.Text()
		blocks = append(blocks, []byte(block))
	}
	return blocks
}

func parseBlock(lexed map[string][]string) *Entry {
	e := &Entry{}

	for k, vs := range lexed {
		switch k {
		case "added":
			e.Added = parseDate(k, vs)
		case "comments":
			e.Comments = parseString(k, vs)
		case "deprecated":
			e.Deprecated = parseDate(k, vs)
		case "description":
			e.Description = vs
		case "macrolanguage":
			e.MacroLanguage = parseString(k, vs)
		case "preferred-value":
			e.PreferredValue = parseString(k, vs)
		case "prefix":
			e.Prefix = vs
		case "scope":
			e.Scope = parseString(k, vs)
		case "subtag":
			e.Subtag = parseString(k, vs)
		case "suppress-script":
			e.SuppressScript = parseScript(k, vs)
		case "tag":
			e.Tag = parseString(k, vs)
		case "type":
			e.Type = parseString(k, vs)
		default:
			log.Fatalf("unexpected key: %q", k)
		}
	}
	return e
}

func parseDate(k string, vs []string) Date {
	if len(vs) != 1 {
		log.Fatalf("key %s has value with length %d != 1", k, len(vs))
	}
	v := vs[0]
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		log.Fatalf("key %s failed parsing value %q: %v", k, v, err)
	}
	return Date(t)
}

func parseScript(k string, vs []string) Script {
	if len(vs) != 1 {
		log.Fatalf("key %s has value with length %d != 1", k, len(vs))
	}
	v := vs[0]
	if len(v) != 4 {
		log.Fatalf("key %s has language with len != 4: %q", k, v)
	}
	// Script codes are in ASCII.
	fixed := [4]rune{}
	for i := 0; i < len(v); i++ {
		fixed[i] = rune(v[i])
	}
	return fixed
}

func parseString(k string, vs []string) string {
	if len(vs) != 1 {
		log.Fatalf("key %s has value with length %d != 1", k, len(vs))
	}
	v := vs[0]
	return v
}

func main() {
	bss := loadBlocks()
	log.Printf("%d blocks in registry", len(bss))

	r := initRegistry(bss)
	for _, bs := range bss[1:] {
		e := parseBlock(lexBlock(string(bs)))
		r.Entries = append(r.Entries, *e)
	}
	e := yaml.NewEncoder(os.Stdout)
	e.Encode(r)
}

package markdown

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/honmaple/snow/builder/page"
	"github.com/honmaple/snow/config"
	"github.com/pelletier/go-toml/v2"
	"github.com/russross/blackfriday/v2"
	"gopkg.in/yaml.v3"
)

var (
	// 兼容hugo
	MARKDOWN_LINE = regexp.MustCompile(`^[-|\+]{3}\s*$`)
	MARKDOWN_MORE = regexp.MustCompile(`^\s*(?i:<!--more-->)\s*$`)
	MARKDOWN_META = regexp.MustCompile(`^([^:]+):(\s+(.*)|$)`)
)

type markdown struct {
	conf config.Config
}

func readMeta(r io.Reader, content *bytes.Buffer, summary *bytes.Buffer) (page.Meta, error) {
	var (
		isMeta    = true
		isFormat  = true
		isSummery = true
		meta      = make(page.Meta)
		scanner   = bufio.NewScanner(r)
	)

	for scanner.Scan() {
		line := scanner.Text()
		if isFormat && MARKDOWN_LINE.MatchString(line) {
			var b bytes.Buffer
			for scanner.Scan() {
				l := scanner.Text()
				if strings.TrimSpace(l) == "" || MARKDOWN_LINE.MatchString(l) {
					break
				}
				b.WriteString(l)
				b.WriteString("\n")
			}
			var (
				err error
				// 不要直接使用meta反序列化数据, 否则子元素map类型也会是page.Meta
				mm = make(map[string]interface{})
			)
			if line == "---" {
				err = yaml.Unmarshal(b.Bytes(), &mm)
			} else {
				err = toml.Unmarshal(b.Bytes(), &mm)
			}
			if err != nil {
				return nil, err
			}
			meta = page.Meta(mm)
			isFormat = false
			continue
		}
		isFormat = false

		if isMeta {
			if match := MARKDOWN_META.FindStringSubmatch(line); match != nil {
				meta.Set(strings.ToLower(match[1]), strings.TrimSpace(match[3]))
				continue
			}
		}
		isMeta = false
		if isSummery && MARKDOWN_MORE.MatchString(line) {
			summary.WriteString(content.String())
			isSummery = false
		}
		content.WriteString(line)
		content.WriteString("\n")
	}
	meta.Done()
	return meta, nil
}

func (m *markdown) Read(r io.Reader) (page.Meta, error) {
	var (
		summary bytes.Buffer
		content bytes.Buffer
	)
	meta, err := readMeta(r, &content, &summary)
	if err != nil {
		return nil, err
	}
	buf := content.Bytes()
	if summary.Len() == 0 {
		meta["summary"] = m.HTML(buf, true)
	} else {
		meta["summary"] = m.HTML(summary.Bytes(), false)
	}
	meta["content"] = m.HTML(buf, false)
	return meta, nil
}

func (m *markdown) HTML(data []byte, summary bool) string {
	d := blackfriday.Run(data, blackfriday.WithRenderer(NewChromaRenderer(m.conf.GetHighlightStyle())))
	if summary {
		return m.conf.GetSummary(string(d))
	}
	return string(d)
}

func New(conf config.Config) page.Reader {
	return &markdown{conf}

}

func init() {
	page.Register(".md", New)
}

package page

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/honmaple/snow/utils"
)

type (
	Section struct {
		Meta         Meta
		Lang         string
		File         string
		Path         string
		Permalink    string
		Slug         string
		Title        string
		Content      string
		Pages        Pages
		HiddenPages  Pages
		SectionPages Pages
		Assets       []string
		Parent       *Section
		Children     Sections
	}
	Sections []*Section
)

func (sec *Section) canWrite() bool {
	return sec.Meta.GetString("path") != ""
}

func (sec *Section) realPath(pathstr string) string {
	return utils.StringReplace(pathstr,
		map[string]string{
			"{section}":      sec.RealName(),
			"{section:slug}": sec.Slug,
		})
}

func (sec *Section) isRoot() bool {
	return sec.Parent == nil
}

func (sec *Section) isEmpty() bool {
	return len(sec.Children) == 0 && len(sec.Pages) == 0 && len(sec.HiddenPages) == 0 && len(sec.SectionPages) == 0
}

func (sec *Section) Paginator() []*paginator {
	return sec.Pages.Filter(sec.Meta.GetString("paginate_filter")).Paginator(
		sec.Meta.GetInt("paginate"),
		sec.Path,
		sec.Meta.GetString("paginate_path"),
	)
}

func (sec *Section) Root() *Section {
	if sec.Parent == nil {
		return sec
	}
	return sec.Parent.Root()
}

func (sec *Section) RealName() string {
	if sec.Parent == nil || sec.Parent.Parent == nil {
		return sec.Title
	}
	return fmt.Sprintf("%s/%s", sec.Parent.RealName(), sec.Title)
}

func (sec *Section) FirstName() string {
	if sec.Parent == nil || sec.Parent.Parent == nil {
		return sec.Title
	}
	return sec.Parent.FirstName()
}

func (secs Sections) setSort(key string) {
	sort.SliceStable(secs, utils.Sort(key, func(k string, i int, j int) int {
		switch k {
		case "-":
			return 0 - strings.Compare(secs[i].Title, secs[j].Title)
		case "title":
			return strings.Compare(secs[i].Title, secs[j].Title)
		case "count":
			return utils.Compare(len(secs[i].Pages), len(secs[j].Pages))
		case "weight":
			return utils.Compare(secs[i].Meta.GetInt("weight"), secs[j].Meta.GetInt("weight"))
		default:
			return 0
		}
	}))
}

func (secs Sections) OrderBy(key string) Sections {
	newSecs := make(Sections, len(secs))
	copy(newSecs, secs)

	newSecs.setSort(key)
	return newSecs
}

func (b *Builder) findSectionIndex(path string, lang string) string {
	prefix := "_index"
	if !b.conf.IsDefaultLanguage(lang) {
		prefix = prefix + "." + lang
	}
	if files := b.findFiles(path, prefix+".*"); len(files) > 0 {
		return files[0]
	}
	return ""
}

func (b *Builder) insertSection(path string, lang string) *Section {
	filemeta := make(Meta)
	if index := b.findSectionIndex(path, lang); index != "" {
		filemeta, _ = b.readFile(index)
	}

	section := &Section{
		Lang:   lang,
		File:   path,
		Parent: b.ctx.findSection(filepath.Dir(path), lang),
	}
	// 根目录
	if section.isRoot() {
		section.Meta = make(Meta)
		section.Meta.load(b.conf.GetStringMap("sections._default"))
	} else {
		section.Meta = section.Parent.Meta.clone()
		section.Title = filepath.Base(section.File)
	}
	section.Meta.load(filemeta)

	name := section.RealName()
	if !section.isRoot() {
		section.Meta.load(b.conf.GetStringMap("sections." + name))
		if !b.conf.IsDefaultLanguage(lang) {
			section.Meta.load(b.conf.GetStringMap("languages." + lang + ".sections." + name))
		}
	}

	for k, v := range section.Meta {
		switch strings.ToLower(k) {
		case "title":
			section.Title = v.(string)
		case "content":
			section.Content = v.(string)
		}
	}

	slug := section.Meta.GetString("slug")
	if slug == "" {
		names := strings.Split(name, "/")
		slugs := make([]string, len(names))
		for i, name := range names {
			slugs[i] = b.conf.GetSlug(name)
		}
		slug = strings.Join(slugs, "/")
	}
	section.Slug = slug
	section.Path = b.conf.GetRelURL(section.realPath(section.Meta.GetString("path")), lang)
	section.Permalink = b.conf.GetURL(section.Path)

	b.ctx.insertSection(section)
	return section
}

func (b *Builder) writeSection(section *Section) {
	if section.canWrite() {
		lookups := []string{
			section.realPath(section.Meta.GetString("template")),
			"section.html",
			"_default/section.html",
		}
		if tpl := b.theme.LookupTemplate(lookups...); tpl != nil {
			for _, por := range section.Paginator() {
				b.write(tpl, por.URL, map[string]interface{}{
					"section":       section,
					"paginator":     por,
					"pages":         section.Pages,
					"current_lang":  section.Lang,
					"current_path":  por.URL,
					"current_index": por.PageNum,
				})
			}
		}
	}
	b.writeFormats(section.Meta, section.realPath, map[string]interface{}{
		"section":      section,
		"pages":        section.Pages,
		"current_lang": section.Lang,
	})
}

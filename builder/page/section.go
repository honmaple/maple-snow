package page

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/honmaple/snow/utils"
)

type (
	Section struct {
		// slug:
		// weight:
		// aliases:
		// transparent:
		// filter:
		// orderby:
		// paginate:
		// paginate_path: {name}{number}{extension}
		// path:
		// template:
		// page_path:
		// page_template:
		// feed_path:
		// feed_template:
		File         string
		Meta         Meta
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
		Lang         string
	}
	Sections []*Section
)

func (sec *Section) vars() map[string]string {
	return map[string]string{"{section}": sec.Name(), "{section:slug}": sec.Slug}
}

func (sec *Section) isRoot() bool {
	return sec.Parent == nil
}

func (sec *Section) isEmpty() bool {
	return len(sec.Children) == 0 && len(sec.Pages) == 0 && len(sec.HiddenPages) == 0 && len(sec.SectionPages) == 0
}

func (sec *Section) isPaginate() bool {
	return sec.Meta.GetInt("paginate") > 0
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

func (sec *Section) Name() string {
	if sec.Parent == nil || sec.Parent.Parent == nil {
		return sec.Title
	}
	return fmt.Sprintf("%s/%s", sec.Parent.Name(), sec.Title)
}

func (sec *Section) FirstName() string {
	if sec.Parent == nil || sec.Parent.Title == "" {
		return sec.Title
	}
	return sec.Parent.FirstName()
}

func (b *Builder) findSectionIndex(prefix string, files map[string]bool) string {
	for ext := range b.readers {
		file := prefix + ext
		if files[file] {
			return file
		}
	}
	return ""
}

func (b *Builder) insertSection(path string) *Section {
	names, _ := utils.FileList(path)
	namem := make(map[string]bool)
	for _, name := range names {
		namem[name] = true
	}

	b.ignoreFiles = b.ignoreFiles[:0]

	b.languageRange(func(lang string, isdefault bool) {
		prefix := "_index"
		if !isdefault {
			prefix = prefix + "." + lang
		}
		filemeta := make(Meta)
		if index := b.findSectionIndex(prefix, namem); index != "" {
			filemeta, _ = b.readFile(filepath.Join(path, index))
		}

		section := &Section{
			File: path,
			Lang: lang,
		}
		section.Parent = b.ctx.findSection(filepath.Dir(section.File), lang)
		// 根目录
		if section.isRoot() {
			section.Meta = make(Meta)
			section.Meta.load(b.conf.GetStringMap("sections._default"))
		} else {
			section.Meta = section.Parent.Meta.clone()
			section.Title = filepath.Base(section.File)
		}
		section.Meta.load(filemeta)

		name := section.Name()
		if !section.isRoot() {
			section.Meta.load(b.conf.GetStringMap("sections." + name))
			if !isdefault {
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
		section.Path = b.conf.GetRelURL(utils.StringReplace(section.Meta.GetString("path"), section.vars()), lang)
		section.Permalink = b.conf.GetURL(section.Path)

		b.ctx.insertSection(section)

		ignoreFiles := filemeta.GetSlice("ignore_files")
		for _, file := range ignoreFiles {
			re, err := regexp.Compile(filepath.Join(path, file))
			if err == nil {
				b.ignoreFiles = append(b.ignoreFiles, re)
			}
		}
	})
	return nil
}

func (b *Builder) writeSection(section *Section) {
	var (
		vars = section.vars()
	)
	if section.Meta.GetString("path") != "" {
		lookups := []string{
			utils.StringReplace(section.Meta.GetString("template"), vars),
			"section.html",
			"_default/section.html",
		}
		if tpl, ok := b.theme.LookupTemplate(lookups...); ok {
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
	b.writeFormats(section.Meta, vars, map[string]interface{}{
		"section":      section,
		"pages":        section.Pages,
		"current_lang": section.Lang,
	})
}

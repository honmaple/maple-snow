package page

import (
	"fmt"
	"strings"
	"sync"

	"github.com/honmaple/snow/builder/theme/template"
	"github.com/honmaple/snow/utils"
	"github.com/panjf2000/ants/v2"
	"github.com/spf13/viper"
)

type taskPool struct {
	*ants.PoolWithFunc
	wg *sync.WaitGroup
}

func (p *taskPool) Invoke(i interface{}) {
	p.wg.Add(1)
	p.PoolWithFunc.Invoke(i)
}

func (p *taskPool) Wait() {
	p.wg.Wait()
}

func newTaskPool(wg *sync.WaitGroup, size int, f func(interface{})) *taskPool {
	p, _ := ants.NewPoolWithFunc(size, f)
	return &taskPool{
		PoolWithFunc: p,
		wg:           wg,
	}
}

func (b *Builder) getSection(name string) *Section {
	for _, section := range b.sections {
		if name == section.Name() {
			return section
		}
	}
	return nil
}

func (b *Builder) getSectionURL(name string) string {
	if sec := b.getSection(name); sec != nil {
		return sec.Permalink
	}
	return ""
}

func (b *Builder) getTaxonomy(name string) *Taxonomy {
	for _, taxonomy := range b.taxonomies {
		if name == taxonomy.Name {
			return taxonomy
		}
	}
	return nil
}

func (b *Builder) getTaxonomyTerm(kind, name string) *TaxonomyTerm {
	for _, taxonomy := range b.taxonomies {
		if kind != taxonomy.Name {
			continue
		}
		if result := taxonomy.Terms.Find(name); result != nil {
			return result
		}
	}
	return nil
}

func (b *Builder) getTaxonomyURL(kind string, names ...string) string {
	if len(names) >= 1 {
		if term := b.getTaxonomyTerm(kind, names[0]); term != nil {
			return term.Permalink
		}
		return ""
	}
	taxonomy := b.getTaxonomy(kind)
	if taxonomy != nil {
		return taxonomy.Permalink
	}
	return ""
}

func (b *Builder) write(tpl template.Writer, path string, vars map[string]interface{}) {
	if path == "" {
		return
	}
	rvars := map[string]interface{}{
		"pages":            b.pages,
		"taxonomies":       b.taxonomies,
		"get_section":      b.getSection,
		"get_section_url":  b.getSectionURL,
		"get_taxonomy":     b.getTaxonomy,
		"get_taxonomy_url": b.getTaxonomyURL,
		"current_url":      b.conf.GetURL(path),
		"current_path":     b.conf.GetRelURL(path),
		"current_template": tpl.Name(),
	}
	for k, v := range rvars {
		if _, ok := vars[k]; !ok {
			vars[k] = v
		}
	}
	// 支持uglyurls和非uglyurls形式
	if strings.HasSuffix(path, "/") {
		path = path + "index.html"
	}
	if err := tpl.Write(path, vars); err != nil {
		b.conf.Log.Error(err.Error())
	}
}

func (b *Builder) writePage(page *Page) {
	if !page.isSection() {
		if tpl, ok := b.theme.LookupTemplate(page.Meta.GetString("template")); ok {
			b.write(tpl, page.Path, map[string]interface{}{
				"page": page,
			})
		}
		if tpl, ok := b.theme.LookupTemplate("aliase.html", "_internal/aliase.html"); ok {
			for _, aliase := range page.Aliases {
				b.write(tpl, aliase, map[string]interface{}{
					"page": page,
				})
			}
		}
		return
	}

	path := page.Meta.GetString("path")
	if path == "" {
		return
	}
	section := &Section{
		Meta:    page.Meta,
		Title:   page.Title,
		Content: page.Content,
		Pages:   page.Section.allPages(),
	}
	section.Slug = b.conf.GetSlug(section.Title)
	section.Path = b.conf.GetRelURL(path)
	section.Permalink = b.conf.GetURL(path)
	section.Pages = section.Pages.Filter(page.Meta.GetString("filter")).OrderBy(page.Meta.GetString("orderby"))

	b.writeSection(section)
}

func (b *Builder) writeSection(section *Section) {
	var (
		vars     = section.vars()
		path     = utils.StringReplace(section.Meta.GetString("path"), vars)
		template = utils.StringReplace(section.Meta.GetString("template"), vars)
	)

	if path != "" {
		if tpl, ok := b.theme.LookupTemplate(template, "section.html", "_internal/section.html"); ok {
			pors := section.Paginator()
			for _, por := range pors {
				b.write(tpl, por.URL, map[string]interface{}{
					"section":       section,
					"paginator":     por,
					"pages":         section.Pages,
					"current_index": por.PageNum,
				})
			}
		}
	}
	b.writeFormats(section.Meta, vars, map[string]interface{}{
		"section": section,
		"pages":   section.Pages,
	})
}

func (b *Builder) writeTaxonomy(taxonomy *Taxonomy) {
	var (
		vars     = taxonomy.vars()
		path     = utils.StringReplace(taxonomy.Meta.GetString("path"), vars)
		template = utils.StringReplace(taxonomy.Meta.GetString("template"), vars)
	)
	if path != "" {
		if tpl, ok := b.theme.LookupTemplate(template,
			fmt.Sprintf("%s/taxonomy.html", taxonomy.Name),
			"taxonomy.html",
			"_default/taxonomy.html",
			"_internal/taxonomy.html",
		); ok {
			// example.com/tags/index.html
			b.write(tpl, path, map[string]interface{}{
				"taxonomy": taxonomy,
				"terms":    taxonomy.Terms,
			})
		}
	}
}

func (b *Builder) writeTaxonomyTerm(term *TaxonomyTerm) {
	var (
		vars         = term.vars()
		termPath     = utils.StringReplace(term.Meta.GetString("term_path"), vars)
		termTemplate = utils.StringReplace(term.Meta.GetString("term_template"), vars)
	)
	if termPath != "" {
		if tpl, ok := b.theme.LookupTemplate(termTemplate,
			fmt.Sprintf("%s/taxonomy.terms.html", term.Taxonomy.Name),
			"taxonomy.terms.html",
			"_default/taxonomy.terms.html",
			"_internal/taxonomy.terms.html",
		); ok {
			pors := term.Paginator()
			for _, por := range pors {
				b.write(tpl, por.URL, map[string]interface{}{
					"term":          term,
					"pages":         term.List,
					"taxonomy":      term.Taxonomy,
					"paginator":     por,
					"current_index": por.PageNum,
				})
			}
		}
	}
	b.writeFormats(term.Meta, vars, map[string]interface{}{
		"term":     term,
		"pages":    term.List,
		"taxonomy": term.Taxonomy,
	})
}

// write rss, atom, json
func (b *Builder) writeFormats(meta Meta, pathvars map[string]string, vars map[string]interface{}) {
	formats := meta.GetStringMap("formats")
	if formats == nil {
		return
	}

	conf := viper.New()
	conf.MergeConfigMap(formats)

	dconf := b.conf.Sub("formats")
	for _, k := range dconf.AllKeys() {
		if !conf.IsSet(k) {
			conf.Set(k, dconf.Get(k))
		}
	}

	for name := range formats {
		var (
			path     = utils.StringReplace(conf.GetString(name+".path"), pathvars)
			template = utils.StringReplace(conf.GetString(name+".template"), pathvars)
		)
		if path == "" || template == "" {
			continue
		}
		if tpl, ok := b.theme.LookupTemplate(template); ok {
			b.write(tpl, path, vars)
		}
	}
}

func (b *Builder) writePages(pages Pages) {
	for _, page := range pages {
		b.tasks.Invoke(page)
	}
}

func (b *Builder) writeSections(sections Sections) {
	for _, section := range sections {
		b.tasks.Invoke(section)
	}
}

func (b *Builder) writeTaxonomies(taxonomies Taxonomies) {
	for _, taxonomy := range taxonomies {
		b.tasks.Invoke(taxonomy)
		b.writeTaxonomyTerms(taxonomy.Terms)
	}
}

func (b *Builder) writeTaxonomyTerms(terms TaxonomyTerms) {
	for _, term := range terms {
		b.tasks.Invoke(term)
		b.writeTaxonomyTerms(term.Children)
	}
}

func (b *Builder) Write() error {
	var wg sync.WaitGroup

	tasks := newTaskPool(&wg, 100, func(i interface{}) {
		switch v := i.(type) {
		case *Page:
			b.writePage(v)
		case *Section:
			b.writeSection(v)
		case *Taxonomy:
			b.writeTaxonomy(v)
		case *TaxonomyTerm:
			b.writeTaxonomyTerm(v)
		}
		wg.Done()
	})
	defer tasks.Release()

	b.tasks = tasks

	b.writePages(b.hooks.BeforePagesWrite(b.pages))
	b.writePages(b.hooks.BeforePagesWrite(b.hiddenPages))
	b.writePages(b.hooks.BeforePagesWrite(b.sectionPages))

	b.writeSections(b.hooks.BeforeSectionsWrite(b.sections))
	b.writeTaxonomies(b.hooks.BeforeTaxonomiesWrite(b.taxonomies))

	tasks.Wait()
	return nil
}

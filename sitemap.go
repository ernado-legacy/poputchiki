package main

import (
	"bytes"
	"fmt"
)

const (
	header = `<?xml version="1.0" encoding="UTF-8"?>
	<urlset xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
	xsi:schemaLocation="http://www.sitemaps.org/schemas/sitemap/0.9 http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd"
	xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
	footer          = ` </urlset>`
	sitemapTemplate = `
	 <url>
	   <loc>%s</loc>
	 </url> 	`

	indexHeader = `<?xml version="1.0" encoding="UTF-8"?>
      <sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`
	indexFooter = `
</sitemapindex>
	`
)

type SitemapItem struct {
	Loc string
}

func (item SitemapItem) String() string {
	return fmt.Sprintf(sitemapTemplate, item.Loc)
}

func SitemapStr(items []SitemapItem) (string, error) {
	var buffer bytes.Buffer
	buffer.WriteString(header)
	for _, item := range items {
		_, err := buffer.WriteString(item.String())
		if err != nil {
			return "", err
		}
	}
	buffer.WriteString(footer)
	return buffer.String(), nil
}

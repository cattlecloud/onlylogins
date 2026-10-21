// SPDX-License-Identifier: AGPL-3.0
// Copyright (c) 2026 Cattle Cloud LLC

package ui

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"

	"cattlecloud.net/onlylogins/pkg/format"
)

// FileTree contains the HTML templates and CSS files that are used to render
// HTML pages. These are loaded via virtual file system into templates that
// can be executed in page endpoint handlers.
//
//go:embed html/*
//go:embed css/*
//go:embed images/*
var FileTree embed.FS

// HTMX contains templates for components rendered dynamically via htmx.
//
//go:embed parts/*
var HTMX embed.FS

// Load is for loading standard pages as template files, with additional basic
// templates and functions used for rendering the complex html of the site.
//
// Additional component parts can be listed to include in the template; useful
// for reusing HTMX components between main page and internal HTMX helper
// handlers.
func Load(filename string, parts ...string) *template.Template {
	// all pages stem from these basic template files
	//
	// further customization is added by loading in the given filename
	basics := []string{
		"layout.html",
		"head.html",
		"foot.html",
	}

	// load our files into virtual filesystem using Go embed
	files, ferr := fs.Sub(FileTree, "html")
	if ferr != nil {
		panic(fmt.Sprintf("ui: unable to load html static files: %v", ferr))
	}

	// configure custom functions for formatting text
	funcs := template.FuncMap{
		"Icon": format.Icon,
	}

	// parse template files
	filenames := append([]string{filename}, basics...)
	t, terr := template.New("root").Funcs(funcs).ParseFS(files, filenames...)
	if terr != nil {
		panic(fmt.Sprintf("ui: unable to load template file: %v", terr))
	}

	// include htmx component parts if necessary
	if len(parts) > 0 {
		pfiles, pferr := fs.Sub(HTMX, "parts")
		if pferr != nil {
			panic(fmt.Sprintf("ui: unable to load parts template files: %v", pferr))
		}

		_, pterr := t.ParseFS(pfiles, parts...)
		if pterr != nil {
			panic(fmt.Sprintf("ui: unable to parse parts template files: %v", pterr))
		}
	}

	// return the desired compiled template
	return t.Lookup(filename)
}

// Mod is for loading moderation pages only.
func Mod(filename string) *template.Template {
	// all moderation pages stem from the same basic template files
	//
	// further customization is added by loading the given filename, one
	// for each page
	basics := []string{
		"mod_layout.html",
	}

	files, ferr := fs.Sub(FileTree, "html")
	if ferr != nil {
		panic(fmt.Sprintf("ui: unable to load moderation files: %v", ferr))
	}

	// configure custom functions for formatting text
	funcs := template.FuncMap{
		"Ago": format.Ago,
	}

	filenames := append([]string{filename}, basics...)
	t, terr := template.New("root").Funcs(funcs).ParseFS(files, filenames...)
	if terr != nil {
		panic(fmt.Sprintf("ui: unable to load moderation template file: %v", terr))
	}

	return t.Lookup(filename)
}

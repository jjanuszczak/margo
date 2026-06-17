package scaffold

import "embed"

//go:embed assets/katex/katex.min.css assets/katex/katex.min.js assets/katex/fonts/*
var defaultThemeMathAssets embed.FS

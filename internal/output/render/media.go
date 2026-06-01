package render

import "regexp"

var leadImagePattern = regexp.MustCompile(`(?s)<p><img[^>]*></p>`)

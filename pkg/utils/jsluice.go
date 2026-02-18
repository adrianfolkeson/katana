//go:build !(386 || windows)

package utils

import (
	"regexp"
	"strings"
)

// Pure Go implementation for extracting endpoints from JavaScript code
// This replaces the jsluice dependency that requires CGO via go-tree-sitter
// Eliminates the need for CGO and enables cross-compilation

var (
	// CommonJSLibraryFileRegex is a regex to match common js library files.
	CommonJSLibraryFileRegex = `(?i)(?:amplify|quantserve|slideshow|jquery|modernizr|polyfill|vendor|modules|gtm|underscore?|tween|retina|selectivizr|cufon|angular|swf|sha1|freestyle|bootstrap|d3|backbone|videojs|google[-_]analytics|material|redux|knockout|datepicker|datetimepicker|ember|react|ng|fusion|analytics|libs?|vendors?|node[-_]modules|lodash|moment|chart|highcharts|raphael|prototype|mootools|dojo|ext|yui|web[-_]?components|polymer|vue|svelte|next|nuxt|gatsby|express|koa|hapi|socket[-_.]?io|axios|superagent|request|bluebird|rxjs|ramda|immutable|flux|redux[-_]saga|mobx|relay|apollo|graphql|three|phaser|pixi|babylon|cannon|hammer|howler|gsap|velocity|mo[-_.]?js|popper|shepherd|prism|highlight|markdown[-_]?it|codemirror|ace[-_]?editor|tinymce|ckeditor|quill|simplemde|monaco[-_]?editor|pdf[-_.]?js|jspdf|fabric|paper|konva|p5|processing|matter[-_.]?js|box2d|planck|chart[-_.]?js|plotly|echarts|d3[-_.]?force|sigma|c3|nvd3|amcharts|vis[-_.]?js|dagre[-_.]?d3|cytoscape|leaflet|openlayers|ol3|mapbox|cesium|turf|moment[-_.]?timezone|luxon|dayjs|date[-_.]?fns|date[-_.]?io|flatpickr|pikaday|fullcalendar|draggable|interact|sortable|dragula|dropzone|filepond|uppy|fine[-_.]?uploader|plyr|mediaelement|flowplayer|jwplayer|video[-_.]?js|mediaelement[-_.]?js|dash[-_.]?js|hls[-_.]?js|videojs|wavesurfer|soundmanager|amplitude|pizzicato|tone|adroll|doubleclick|facebook-pixel|ga-audiences|googlesyndication|adsbygoogle|gpt|amazon-adsystem|criteo|taboola|outbrain|bidswitch|bidswitch.net|spotxchange|yahoo|media.net|contextweb|openx|pubmatic|rubiconproject|indexexchange|appnexus|liveintent|triplelift|verizonmedia|synacor|sonobi|yieldmo|gumgum|smartadserver|mopub|pubnative|inmobi|chartboost|tapjoy|admob|unityads|vungle|flurry|matomy|altitude|dataxu|thetradedesk|exponential|zypmedia|quantcast|mediamath|bidswitch|mgid|revcontent|powerlinks|rhythmone|airpush|smaato|adcolony|mopub|leadbolt|mobfox|nativo|revjet|smartyads|avocarrot|epom|imobile|supersonicads|loopme|applovin|pandora|mytarget|bidvertiser|chitika|popads|propellerads|buysellads|adhit|hilltopads|plugrush|popcash|popunder|revenuehits|trafficjunky|trafficfactory|zero-|smartoasis)(?:[-._][\w\d]*)*\.js$`
	commonJSLibraryFileRegexCompiled = regexp.MustCompile(CommonJSLibraryFileRegex)

	// Regex patterns for matching URLs in JavaScript
	// These patterns match common ways URLs appear in JavaScript code
	jsURLPatterns = []*regexp.Regexp{
		// Match strings in quotes that look like URLs
		regexp.MustCompile(`['"]https?://[^'"<>]+[^'"<>.,)\s]`),
		regexp.MustCompile(`['"]/[^'"<>]+[^'"<>.,)\s]`),

		// Match template literals with URLs
		regexp.MustCompile("```https?://[^`<>]+"),
		regexp.MustCompile("`` `/[^`<>]+`"),

		// Match fetch/axios/XMLHttpRequest calls
		regexp.MustCompile(`fetch\s*\(\s*['"`+`"`+`](https?://[^'"<>]+)`),
		regexp.MustCompile(`axios\s*\.\s*(?:get|post|put|delete|patch)\s*\(\s*['"`+`"`+`](https?://[^'"<>]+)`),
		regexp.MustCompile(`\$\.(?:get|post|ajax)\s*\(\s*['"`+`"`+`](https?://[^'"<>]+)`),
		regexp.MustCompile(`XMLHttpRequest\s*\.\s*open\s*\([^,]+,\s*['"`+`"`+`](https?://[^'"<>]+)`),

		// Match API endpoint patterns (REST, GraphQL)
		regexp.MustCompile(`['"`+`"`+`](?:/api/|/v\d+/|/graphql|/ws/|/rest/)[^'"<>]*`),

		// Match common URL patterns in object properties
		regexp.MustCompile(`(?:baseURL|baseUrl|endpoint|apiUrl|api_url|url|URL)\s*[:=]\s*['"`+`"`+`](https?://[^'"<>]+)`),
		regexp.MustCompile(`(?:baseURL|baseUrl|endpoint|apiUrl|api_url|url|URL)\s*[:=]\s*['"`+`"`+`](/[^'"<>]+)`),

		// Match URLs in comments (sometimes developers document endpoints there)
		regexp.MustCompile(`//\s*(https?://[^\s]+)`),
		regexp.MustCompile(`/\\*\s*(https?://[^\s]+)\s*\\*/`),
	}
)

// IsPathCommonJSLibraryFile checks if a given path is a common js library file.
func IsPathCommonJSLibraryFile(path string) bool {
	return commonJSLibraryFileRegexCompiled.MatchString(path)
}

type JSLuiceEndpoint struct {
	Endpoint string
	Type     string
}

// ExtractJsluiceEndpoints extracts endpoints from JavaScript code using pure Go
// This is a CGO-free replacement for jsluice that enables cross-compilation
//
// The function uses regex patterns to identify:
// - URLs in string literals
// - Fetch, Axios, and XHR API calls
// - REST API endpoints
// - GraphQL endpoints
// - WebSocket connections
// - URLs in comments
func ExtractJsluiceEndpoints(data string) []JSLuiceEndpoint {
	seen := make(map[string]bool)
	var endpoints []JSLuiceEndpoint

	lines := strings.Split(data, "\n")

	for lineNum, line := range lines {
		// Skip very long lines (likely minified)
		if len(line) > 10000 {
			continue
		}

		// Try each pattern
		for _, pattern := range jsURLPatterns {
			matches := pattern.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				var url string
				if len(match) > 1 {
					url = match[1]
				} else {
					url = match[0]
				}

				// Clean up the URL
				url = cleanURL(url)
				if url == "" || seen[url] {
					continue
				}

				// Skip common library URLs
				if isCommonLibraryURL(url) {
					continue
				}

				// Determine the type
				endpointType := determineEndpointType(line, url, lineNum)

				seen[url] = true
				endpoints = append(endpoints, JSLuiceEndpoint{
					Endpoint: url,
					Type:     endpointType,
				})
			}
		}
	}

	return endpoints
}

// cleanURL removes extra characters from extracted URLs
func cleanURL(url string) string {
	url = strings.TrimSpace(url)

	// Remove trailing quotes, commas, semicolons, etc.
	url = strings.TrimRight(url, `'",;:})]`)

	// Remove leading quotes
	url = strings.TrimLeft(url, `'"`)

	// Remove template literal markers
	url = strings.Trim(url, "`")

	// If it's just a path without protocol, make sure it starts with /
	if len(url) > 0 && url[0] != '/' &&
		!strings.HasPrefix(url, "http://") &&
		!strings.HasPrefix(url, "https://") &&
		!strings.HasPrefix(url, "ws://") &&
		!strings.HasPrefix(url, "wss://") {
		// Check if it looks like a path
		if strings.Contains(url, "/") {
			url = "/" + url
		} else {
			return ""
		}
	}

	// Basic validation
	if len(url) < 2 || url == "/" || url == "//" {
		return ""
	}

	return url
}

// determineEndpointType determines the type of endpoint based on context
func determineEndpointType(line, url string, lineNum int) string {
	lowerLine := strings.ToLower(line)

	// Check for websocket URLs
	if strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://") {
		return "websocket"
	}

	// Check for graphql
	if strings.Contains(lowerLine, "graphql") || strings.Contains(line, "query{") || strings.Contains(line, "mutation{") {
		return "graphql"
	}

	// Check for fetch
	if strings.Contains(lowerLine, "fetch(") {
		return "fetch"
	}

	// Check for axios
	if strings.Contains(lowerLine, "axios.") {
		return "axios"
	}

	// Check for XHR
	if strings.Contains(lowerLine, "xmlhttprequest") || strings.Contains(lowerLine, ".open(") {
		return "xhr"
	}

	// Check for jQuery AJAX
	if strings.Contains(lowerLine, "$.get") || strings.Contains(lowerLine, "$.post") || strings.Contains(lowerLine, "$.ajax") {
		return "jquery"
	}

	// Check for API endpoints
	if strings.Contains(lowerLine, "/api/") || strings.Contains(lowerLine, "/v1/") || strings.Contains(lowerLine, "/v2/") {
		return "api"
	}

	// Check for static files
	if strings.Contains(strings.ToLower(url), ".js") ||
		strings.Contains(strings.ToLower(url), ".css") ||
		strings.Contains(strings.ToLower(url), ".png") ||
		strings.Contains(strings.ToLower(url), ".jpg") {
		return "static"
	}

	// Check if it's in a comment
	if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "/*") {
		return "comment"
	}

	// Default to literal
	return "literal"
}

// isCommonLibraryURL checks if a URL is from a common JS library/CDN
func isCommonLibraryURL(url string) bool {
	lowerURL := strings.ToLower(url)

	commonDomains := []string{
		"jquery", "googleapis.com", "gstatic.com", "cloudflare.com",
		"cdnjs", "jsdelivr", "unpkg.com", "cdn.jsdelivr.net",
		"fontawesome", "bootstrap", "angular", "react",
		"vue", "svelte", "nextjs", "analytics",
		"doubleclick", "googletagmanager", "facebook", "fbcdn",
		"googlesyndication", "adsystem",
	}

	for _, domain := range commonDomains {
		if strings.Contains(lowerURL, domain) {
			return true
		}
	}

	return false
}

package main

import (
	"testing"
)

// Verify that the CDN is not manipulating response bodies such as code
// minification or optimisation, lossy or lossless image compression,
// stripping image metadata, etc. We do not want this to happen magically,
// we'd rather do it ourselves.

// Should not manipulate HTML content in response bodies.
func TestNoManipulationHTML(t *testing.T) {
	ResetBackends(backendsByPriority)

	const fixtureFile = "fixtures/golang.html"
	const contentType = "text/html; charset=utf-8"

	testResponseNotManipulated(t, fixtureFile, contentType)
}

// Should not manipulate CSS content in response bodies.
func TestNoManipulationCSS(t *testing.T) {
	ResetBackends(backendsByPriority)

	const fixtureFile = "fixtures/golang.css"
	const contentType = "text/css; charset=utf-8"

	testResponseNotManipulated(t, fixtureFile, contentType)
}

// Should not manipulate Javascript content in response bodies.
func TestNoManipulationJS(t *testing.T) {
	ResetBackends(backendsByPriority)

	const fixtureFile = "fixtures/golang.js"
	const contentType = "application/x-javascript"

	testResponseNotManipulated(t, fixtureFile, contentType)
}

// Should not manipulate PNG images in response bodies.
func TestNoManipulationPNG(t *testing.T) {
	ResetBackends(backendsByPriority)

	const fixtureFile = "fixtures/golang.png"
	const contentType = "image/png"

	testResponseNotManipulated(t, fixtureFile, contentType)
}

// Should not manipulate JPEG images in response bodies.
func TestNoManipulationJPEG(t *testing.T) {
	ResetBackends(backendsByPriority)

	const fixtureFile = "fixtures/golang.jpeg"
	const contentType = "image/jpeg"

	testResponseNotManipulated(t, fixtureFile, contentType)
}

// Should not manipulate GIF images in response bodies.
func TestNoManipulationGIF(t *testing.T) {
	ResetBackends(backendsByPriority)

	const fixtureFile = "fixtures/golang.gif"
	const contentType = "image/gif"

	testResponseNotManipulated(t, fixtureFile, contentType)
}

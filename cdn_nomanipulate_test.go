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

	testResponseNotManipulated(t, "fixtures/golang.html")
}

// Should not manipulate CSS content in response bodies.
func TestNoManipulationCSS(t *testing.T) {
	ResetBackends(backendsByPriority)

	testResponseNotManipulated(t, "fixtures/golang.css")
}

// Should not manipulate JavaScript content in response bodies.
func TestNoManipulationJS(t *testing.T) {
	ResetBackends(backendsByPriority)

	testResponseNotManipulated(t, "fixtures/golang.js")
}

// Should not manipulate PNG images in response bodies.
func TestNoManipulationPNG(t *testing.T) {
	ResetBackends(backendsByPriority)

	testResponseNotManipulated(t, "fixtures/golang.png")
}

// Should not manipulate JPEG images in response bodies.
func TestNoManipulationJPEG(t *testing.T) {
	ResetBackends(backendsByPriority)

	testResponseNotManipulated(t, "fixtures/golang.jpeg")
}

// Should not manipulate GIF images in response bodies.
func TestNoManipulationGIF(t *testing.T) {
	ResetBackends(backendsByPriority)

	testResponseNotManipulated(t, "fixtures/golang.gif")
}

// Package chroma provides a Go wrapper for the Chroma vector database,
// using FFI to communicate with the Chroma runtime via a shared library.
//
// Use Init to load the shared library, then NewServer or NewEmbedded
// to start a Chroma instance.
package chroma

package handlers

import "github.com/gorilla/schema"

// muxDecoder decodes URL query parameters into request structs by their schema
// tags. One shared instance suffices: a schema.Decoder is safe for concurrent
// use once configured.
var muxDecoder = schema.NewDecoder()

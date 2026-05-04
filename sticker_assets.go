package main

import "embed"

const stickerRoot = "resources/stickers"

//go:embed resources/stickers/**
var stickerAssets embed.FS

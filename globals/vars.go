package globals

// Configuration values now live in package config. This alias is kept for
// source compatibility with existing call sites and will be removed once all
// references are migrated.
//
// Deprecated: use angadrive/config and angadrive/storage instead.
const UPLOAD_DIR string = "uploaded_files"

package main

import (
	"context"
	"os"
)

type InstalledApp struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type AppFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type AppScanIssue struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Error string `json:"error"`
}

type AppReview struct {
	App        InstalledApp   `json:"app"`
	BundleID   string         `json:"bundleId"`
	Files      []AppFile      `json:"files"`
	ScanIssues []AppScanIssue `json:"scanIssues"`
	ReviewID   uint64         `json:"reviewId"`
}

type appRemoval struct {
	app       string
	files     map[string]os.FileInfo
	id        uint64
	cancelled bool
	cancel    context.CancelFunc
}

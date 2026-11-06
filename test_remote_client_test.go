package main

import (
	"github.com/soulteary/warden/internal/cmd"
	"github.com/soulteary/warden/internal/loader"
	"net/http"
)

func newTestRulesLoader(cfg *cmd.Config, mode string) (*loader.RulesLoader, error) {
	return loader.NewRulesLoaderWithHTTPClient(cfg, mode, &http.Client{})
}

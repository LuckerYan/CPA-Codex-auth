package management

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/managementasset"
)

const codexExtractionAssetName = "codex-extract.html"

// codexExtractionConfigFilePath returns the active config file path that the
// handler should reference when resolving the static asset directory. The
// handler doesn't store this directly, so we re-read it through the package
// helper that's used by management.html.
var codexExtractionConfigFilePath = func() string { return "" }

// SetCodexExtractionConfigFilePath wires the config file path that
// ServeCodexExtractionPage will use for resolving the static asset directory.
func SetCodexExtractionConfigFilePath(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	codexExtractionConfigFilePath = func() string { return path }
}

func (h *Handler) ServeCodexExtractionPage(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	filePath := resolveCodexExtractionAssetPath()
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		return
	} else if err != nil && !os.IsNotExist(err) {
		log.WithError(err).Warn("failed to read codex extraction asset")
	}

	if h != nil && h.cfg != nil {
		staticDir := managementasset.StaticDir(codexExtractionConfigFilePath())
		if managementasset.EnsureLatestCodexExtractionHTML(
			context.Background(),
			staticDir,
			h.cfg.ProxyURL,
			h.cfg.RemoteManagement.PanelGitHubRepository,
		) {
			if data, err := os.ReadFile(filePath); err == nil && len(data) > 0 {
				c.Data(http.StatusOK, "text/html; charset=utf-8", data)
				return
			}
		}
	}

	c.AbortWithStatus(http.StatusNotFound)
}

func resolveCodexExtractionAssetPath() string {
	staticDir := managementasset.StaticDir(codexExtractionConfigFilePath())
	if strings.TrimSpace(staticDir) == "" {
		return ""
	}
	return filepath.Join(staticDir, codexExtractionAssetName)
}

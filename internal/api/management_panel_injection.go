package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// serveManagementControlPanelAsset reads the on-disk management.html (kept in
// sync from the configured panel-github-repository release) and writes it to
// the response. The asset is now produced by the React fork itself, so the
// server no longer has to monkey-patch the bundle at request time.
func (s *Server) serveManagementControlPanelAsset(c *gin.Context, filePath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.WithError(err).Error("failed to read management control panel asset")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

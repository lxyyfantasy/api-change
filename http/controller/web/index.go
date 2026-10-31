package web

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

type Index struct {
}

func (i *Index) Index(c *gin.Context) {
	c.Redirect(302, "/_admin/")
}

func (i *Index) ConfigJs(c *gin.Context) {
	apiServer := global.Config.Rustdesk.ApiServer
	magicQueryonline := global.Config.Rustdesk.WebclientMagicQueryonline
	host := c.Request.Host
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		host = host[:idx]
	}
	tmp := fmt.Sprintf(`localStorage.setItem('api-server', '%v');
const ws2_prefix = 'wc-';
localStorage.setItem(ws2_prefix+'api-server', '%v');
localStorage.setItem('custom-rendezvous-server', '%v');

window.webclient_magic_queryonline = %d;
window.ws_host = '%v';
`, apiServer, apiServer, host, magicQueryonline, global.Config.Rustdesk.WsHost)

	c.Header("Content-Type", "application/javascript")
	c.String(200, tmp)
}

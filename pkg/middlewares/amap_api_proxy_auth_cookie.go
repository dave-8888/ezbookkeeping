package middlewares

import (
	"github.com/mayswind/financially_free/pkg/core"
	"github.com/mayswind/financially_free/pkg/settings"
)

// AmapApiProxyAuthCookie adds amap api proxy auth cookie to cookies in response
func AmapApiProxyAuthCookie(c *core.WebContext, config *settings.Config) {
	token := c.GetTextualToken()
	c.SetTokenStringToCookie(token, int(config.TokenExpiredTime), "/_AMapService")
}

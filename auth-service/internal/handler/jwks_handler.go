package handler

import (
	"encoding/base64"
	"encoding/binary"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *UserHandler) JWKS(c *gin.Context) {
	pub := h.authSvc.PublicKey()
	n := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())

	eBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(eBuf, uint32(pub.E))
	i := 0
	for i < len(eBuf)-1 && eBuf[i] == 0 {
		i++
	}
	e := base64.RawURLEncoding.EncodeToString(eBuf[i:])

	c.JSON(http.StatusOK, gin.H{"keys": []gin.H{{
		"kty": "RSA", "use": "sig", "alg": "RS256",
		"kid": "auth-service-key-1", "n": n, "e": e,
	}}})
}

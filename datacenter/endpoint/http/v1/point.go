package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kv-iot/datacenter/protocol"
)

func (a *API) testDevicePointBinding(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	bindingID, err := strconv.ParseInt(c.Param("bindingId"), 10, 64)
	if err != nil || bindingID <= 0 {
		writeError(c, http.StatusBadRequest, codeBadRequest, "invalid point binding id")
		return
	}
	if a.resolver == nil {
		writeError(c, http.StatusBadRequest, codeBadRequest, "channel runtime resolver is not configured")
		return
	}
	channelID, point, err := a.resolver.ResolveDevicePointBinding(c.Request.Context(), deviceID, bindingID)
	if err != nil {
		writeError(c, http.StatusBadRequest, codeBadRequest, err.Error())
		return
	}
	startedAt := time.Now()
	samples, err := a.channels.TestChannel(c.Request.Context(), channelID, []protocol.PointBinding{point})
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, gin.H{
		"channel_id":  channelID,
		"duration_ms": time.Since(startedAt).Milliseconds(),
		"samples":     samples,
	}, "point binding test completed")
}

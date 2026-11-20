package v1

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kv-iot/datacenter/runtime"
)

func (a *API) listEvents(c *gin.Context) {
	filter := runtime.EventFilter{
		ChannelID: strings.TrimSpace(c.Query("channelId")),
		Severity:  strings.TrimSpace(c.Query("severity")),
	}
	if value := c.Query("deviceId"); value != "" {
		deviceID, err := strconv.ParseInt(value, 10, 64)
		if err != nil || deviceID <= 0 {
			writeError(c, http.StatusBadRequest, codeBadRequest, "deviceId must be a positive integer")
			return
		}
		filter.DeviceID = deviceID
	}
	if value := c.Query("since"); value != "" {
		since, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "since must be RFC3339")
			return
		}
		filter.Since = since
	}
	if value := c.Query("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit <= 0 {
			writeError(c, http.StatusBadRequest, codeBadRequest, "limit must be a positive integer")
			return
		}
		filter.Limit = limit
	}

	events, err := a.events.List(c.Request.Context(), filter)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, events, "events")
}

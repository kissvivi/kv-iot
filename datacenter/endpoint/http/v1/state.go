package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (a *API) listDeviceState(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	samples, err := a.state.ListByDevice(c.Request.Context(), deviceID)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, samples, "device state")
}

func (a *API) getDeviceState(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	sample, err := a.state.Get(c.Request.Context(), deviceID, c.Param("point"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, sample, "point state")
}

func parseDeviceID(c *gin.Context) (int64, bool) {
	deviceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || deviceID <= 0 {
		writeError(c, http.StatusBadRequest, codeBadRequest, "device id must be a positive integer")
		return 0, false
	}
	return deviceID, true
}

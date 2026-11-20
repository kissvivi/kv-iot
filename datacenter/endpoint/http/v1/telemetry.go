package v1

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kv-iot/datacenter/storage"
)

func (a *API) listTelemetry(c *gin.Context) {
	deviceID, ok := parseDeviceID(c)
	if !ok {
		return
	}
	if a.telemetry == nil {
		writeError(c, http.StatusServiceUnavailable, codeBadRequest, "telemetry reader is not configured")
		return
	}
	query := storage.TelemetryQuery{
		DeviceID:        deviceID,
		PointIdentifier: strings.TrimSpace(c.Query("point_identifier")),
	}
	var err error
	if raw := strings.TrimSpace(c.Query("from")); raw != "" {
		query.From, err = parseTime(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "invalid from time: "+err.Error())
			return
		}
	}
	if raw := strings.TrimSpace(c.Query("to")); raw != "" {
		query.To, err = parseTime(raw)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "invalid to time: "+err.Error())
			return
		}
	}
	if !query.From.IsZero() && !query.To.IsZero() && query.To.Before(query.From) {
		writeError(c, http.StatusBadRequest, codeBadRequest, "to time must not be before from time")
		return
	}
	query.Limit, _ = strconv.Atoi(c.Query("limit"))
	samples, err := a.telemetry.QueryPointSamples(c.Request.Context(), query)
	if err != nil {
		writeError(c, http.StatusBadGateway, codeBadRequest, "query telemetry: "+err.Error())
		return
	}
	writeSuccess(c, http.StatusOK, samples, "telemetry")
}

func parseTime(value string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", value)
}

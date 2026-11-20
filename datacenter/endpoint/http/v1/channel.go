package v1

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"kv-iot/datacenter/protocol"
	"kv-iot/datacenter/runtime"
)

func (a *API) listChannels(c *gin.Context) {
	writeSuccess(c, http.StatusOK, a.channels.List(), "channels")
}

func (a *API) getChannelStatus(c *gin.Context) {
	snapshot, exists := a.channels.Snapshot(c.Param("id"))
	if !exists {
		writeRuntimeError(c, runtime.ErrChannelNotFound)
		return
	}
	writeSuccess(c, http.StatusOK, snapshot, "channel status")
}

func (a *API) startChannel(c *gin.Context) {
	var spec runtime.ChannelSpec
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&spec); err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "invalid channel request: "+err.Error())
			return
		}
	}
	if channelID := strings.TrimSpace(c.Param("id")); channelID != "" {
		spec.ChannelID = channelID
	}
	if len(spec.Config) == 0 {
		if a.resolver == nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "channel runtime resolver is not configured")
			return
		}
		resolved, err := a.resolver.LoadChannelSpec(c.Request.Context(), spec.ChannelID)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, err.Error())
			return
		}
		spec = resolved
	}
	if err := a.channels.StartChannel(c.Request.Context(), spec); err != nil {
		writeRuntimeError(c, err)
		return
	}
	snapshot, _ := a.channels.Snapshot(spec.ChannelID)
	writeSuccess(c, http.StatusCreated, snapshot, "channel started")
}

func (a *API) stopChannel(c *gin.Context) {
	channelID := c.Param("id")
	if err := a.channels.StopChannel(c.Request.Context(), channelID); err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, gin.H{"channel_id": channelID, "status": runtime.ChannelStatusStopped}, "channel stopped")
}

func (a *API) restartChannel(c *gin.Context) {
	channelID := c.Param("id")
	var err error
	if a.resolver != nil {
		var spec runtime.ChannelSpec
		spec, err = a.resolver.LoadChannelSpec(c.Request.Context(), channelID)
		if err == nil {
			err = a.channels.StopChannel(c.Request.Context(), channelID)
			if errors.Is(err, runtime.ErrChannelNotFound) {
				err = nil
			}
		}
		if err == nil {
			err = a.channels.StartChannel(c.Request.Context(), spec)
		}
	} else {
		err = a.channels.RestartChannel(c.Request.Context(), channelID)
	}
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	snapshot, _ := a.channels.Snapshot(channelID)
	writeSuccess(c, http.StatusOK, snapshot, "channel restarted")
}

type channelTestRequest struct {
	Points []protocol.PointBinding `json:"points,omitempty"`
}

func (a *API) testChannel(c *gin.Context) {
	var request channelTestRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, "invalid test request: "+err.Error())
			return
		}
	}
	startedAt := time.Now()
	samples, err := a.channels.TestChannel(c.Request.Context(), c.Param("id"), request.Points)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	writeSuccess(c, http.StatusOK, gin.H{
		"duration_ms": time.Since(startedAt).Milliseconds(),
		"samples":     samples,
	}, "channel test completed")
}

func (a *API) reloadProfileChannels(c *gin.Context) {
	if a.resolver == nil {
		writeError(c, http.StatusBadRequest, codeBadRequest, "channel runtime resolver is not configured")
		return
	}
	profileID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || profileID <= 0 {
		writeError(c, http.StatusBadRequest, codeBadRequest, "invalid profile id")
		return
	}
	channelIDs, err := a.resolver.ListChannelIDsByProfile(c.Request.Context(), profileID)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}
	reloaded := make([]runtime.ChannelSnapshot, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		if _, running := a.channels.Snapshot(channelID); !running {
			continue
		}
		spec, err := a.resolver.LoadChannelSpec(c.Request.Context(), channelID)
		if err != nil {
			writeError(c, http.StatusBadRequest, codeBadRequest, err.Error())
			return
		}
		if err := a.channels.StopChannel(c.Request.Context(), channelID); err != nil {
			writeRuntimeError(c, err)
			return
		}
		if err := a.channels.StartChannel(c.Request.Context(), spec); err != nil {
			writeRuntimeError(c, err)
			return
		}
		snapshot, _ := a.channels.Snapshot(channelID)
		reloaded = append(reloaded, snapshot)
	}
	writeSuccess(c, http.StatusOK, reloaded, "profile channels reloaded")
}

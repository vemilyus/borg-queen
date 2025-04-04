// Copyright (C) 2025 Alex Katlein
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-mods/zerolog-gin"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"github.com/vemilyus/borg-queen/credentials/internal/store/service"
	"net/http"
)

func version(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, state.GetVersion())
	}
}

func unlock(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var passphraseRequest model.PassphraseRequest
		if err := c.ShouldBindJSON(&passphraseRequest); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Message: err.Error()})
			return
		}

		err, ok := state.Unlock(passphraseRequest)
		if !ok {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func lock(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok := state.Lock()
		if !ok {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Message: "already locked"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func SetupEngine(state *service.State) *gin.Engine {
	if state.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	_ = r.SetTrustedProxies([]string{})

	r.Use(zerologgin.Logger())

	// Core endpoints
	r.GET("/version", version(state))
	r.POST("/unlock", unlock(state))
	r.DELETE("/lock", lock(state))

	// Endpoints using passphrase auth
	r.POST("/admin/recovery-recipient", setRecoveryRecipient(state))
	r.GET("/admin/item", readVaultItem(state))
	r.GET("/admin/item/list", listVaultItems(state))
	r.DELETE("/admin/item", deleteVaultItems(state))
	r.POST("/admin/client", createClientCredentials(state))

	// Endpoints using client auth
	r.GET("/client/item", clientReadVaultItems(state))

	return r
}

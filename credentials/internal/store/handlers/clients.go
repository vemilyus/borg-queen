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
	"github.com/awnumar/memguard"
	"github.com/gin-gonic/gin"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"github.com/vemilyus/borg-queen/credentials/internal/store/service"
	"net/http"
)

func createClientCredentials(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var createClientCredentialsRequest model.CreateClientCredentialsRequest
		if err := c.ShouldBindJSON(&createClientCredentialsRequest); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Message: err.Error()})
			return
		}

		response, err := state.CreateClientCredentials(createClientCredentialsRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func clientReadVaultItems(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var clientReadVaultItemRequest model.ClientReadVaultItemRequest
		if err := c.ShouldBindUri(&clientReadVaultItemRequest); err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse{Message: err.Error()})
			return
		}

		response, err := state.ClientReadVaultItem(clientReadVaultItemRequest, c.RemoteIP())
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		defer memguard.WipeBytes(response.Value)

		c.JSON(http.StatusOK, response)
	}
}

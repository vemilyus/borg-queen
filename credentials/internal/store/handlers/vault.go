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
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"github.com/vemilyus/borg-queen/credentials/internal/store/service"
	"net/http"
)

func setRecoveryRecipient(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var setRecoveryRecipientRequest model.SetRecoveryRecipientRequest
		if err := c.ShouldBindJSON(&setRecoveryRecipientRequest); err != nil {
			c.JSON(http.StatusBadRequest, &model.ErrorResponse{Message: err.Error()})
			return
		}

		defer setRecoveryRecipientRequest.Wipe()

		err := state.SetRecoveryRecipient(setRecoveryRecipientRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func createVaultItem(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var createVaultItemRequest model.CreateVaultItemRequest
		if err := c.ShouldBindJSON(&createVaultItemRequest); err != nil {
			c.JSON(http.StatusBadRequest, &model.ErrorResponse{Message: err.Error()})
			return
		}

		defer createVaultItemRequest.Wipe()

		response, err := state.CreateVaultItem(createVaultItemRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func listVaultItems(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var vaultItemsRequest model.ListVaultItemsRequest
		if err := c.ShouldBindQuery(&vaultItemsRequest); err != nil {
			c.JSON(http.StatusBadRequest, &model.ErrorResponse{Message: err.Error()})
			return
		}

		defer vaultItemsRequest.Wipe()

		response, err := state.ListVaultItems(vaultItemsRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

func readVaultItem(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var vaultItemRequest model.ReadVaultItemRequest
		if err := c.ShouldBindQuery(&vaultItemRequest); err != nil {
			c.JSON(http.StatusBadRequest, &model.ErrorResponse{Message: err.Error()})
			return
		}

		defer vaultItemRequest.Wipe()

		response, err := state.ReadVaultItem(vaultItemRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		defer response.Wipe()

		c.JSON(http.StatusOK, response)
	}
}

func deleteVaultItems(state *service.State) gin.HandlerFunc {
	return func(c *gin.Context) {
		var deleteItemRequest model.DeleteVaultItemsRequest
		if err := c.ShouldBindQuery(&deleteItemRequest); err != nil {
			c.JSON(http.StatusBadRequest, &model.ErrorResponse{Message: err.Error()})
			return
		}

		defer deleteItemRequest.Wipe()

		response, err := state.DeleteVaultItems(deleteItemRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, err)
			return
		}

		c.JSON(http.StatusOK, response)
	}
}

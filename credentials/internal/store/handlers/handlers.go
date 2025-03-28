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
	"github.com/gofiber/fiber/v2"
	"github.com/vemilyus/borg-queen/credentials/internal/model"
	"github.com/vemilyus/borg-queen/credentials/internal/store/state"
)

func Setup(app *fiber.App, state *state.State) {
	app.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(model.VersionResponse{
			Version:      state.Version(),
			IsProduction: state.IsProduction(),
		})
	})
}

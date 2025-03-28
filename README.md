<!--
 Copyright (C) 2025 Alex Katlein

 This program is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program. If not, see <https://www.gnu.org/licenses/>.
-->

# borg-queen

Helps to setup and control as many Borg backups as possible with minimal configuration.

## Features/Goals

- Automatically configuring backups for Docker containers using labels
- Connecting directly to Docker daemon to get the required information
- Enabling secure configuration and retrieval of encryption credentials for Borg
- Running as non-privileged user with necessary capabilities configured for Borg

## Non-Goals

- Running inside a container

## Tools

- _borg-queen_ (backup setup on the individual hosts)
- _credential-host_ (securely manages and provides encryption keys for Borg backups)
- _credential-cli_ (interacts securely with _credential-host_ on the individual hosts)

## Target platforms

- linux/amd64
- linux/arm64
- linux/armv7l

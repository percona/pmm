// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// Package highavailability contains everything related to high availability.
package highavailability

import "github.com/AlekSi/pointer"

type Service struct {
	passiveMode bool
}

func New(passiveMode *bool) *Service {
	return &Service{passiveMode: pointer.GetBool(passiveMode)}
}

func (s *Service) ActiveMode() bool {
	return !s.passiveMode
}

func (s *Service) PassiveMode() bool {
	return s.passiveMode
}

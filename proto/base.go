/*
   Copyright (c) 2016, Percona LLC and/or its affiliates. All rights reserved.

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package proto

import (
	"encoding/json"
	"log"
	"net/http"
)

const VERSION = "1.0"

const (
	DEFAULT_AGENT_API_PORT       = "9000"
	DEFAULT_QAN_API_PORT         = "9001"
	DEFAULT_PROM_CONFIG_API_PORT = "9003"
	DEFAULT_METRICS_API_PORT     = "9004"
)

type AuthResponse struct {
	Code  uint   // standard HTTP status (http://httpstatus.es/)
	Error string // empty if auth ok (Code=200)
}

type Error struct {
	Error string
}

func WriteAccessControlHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func JSONResponse(w http.ResponseWriter, statusCode int, v interface{}) {
	WriteAccessControlHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Println(err)
		}
	}
}

func ErrorResponse(w http.ResponseWriter, err error) {
	WriteAccessControlHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(500)
	e := Error{
		Error: err.Error(),
	}
	if err := json.NewEncoder(w).Encode(e); err != nil {
		log.Println(err)
	}
}

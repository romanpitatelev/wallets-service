package rest

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
)

const dateTimeFmt = "01:02:2006 15:04:05"

func (s *Server) TimeNow(w http.ResponseWriter, r *http.Request) {
	ipString, _, _ := net.SplitHostPort(r.RemoteAddr)

	timeCurr, err := s.service.Add(r.Context(), ipString)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err = w.Write([]byte(err.Error())); err != nil {
			log.Error().Err(err).Msg("Internal Server Error")
		}
	}

	_, err = w.Write([]byte(fmt.Sprint("Current date and time: ", timeCurr.Format(dateTimeFmt))))
	if err != nil {
		log.Error().Msg("Write failed")
	}
}

func (s *Server) GetVisitors(w http.ResponseWriter, r *http.Request) {
	visits, err := s.service.GetVisitsAll(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err = w.Write([]byte(err.Error())); err != nil {
			log.Error().Err(err).Msg("Internal Server Error")
		}
	}

	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(visits); err != nil {
		log.Error().Err(err).Msg("Write failed")
	}
}

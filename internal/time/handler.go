package time

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/romanpitatelev/wallets-service/internal/store"
	"github.com/rs/zerolog/log"
)

type TimeHandler struct {
	store *store.VisitorStore
}

func NewTimeHandler(router *chi.Mux) {
	visitorStore := store.NewVisitorStore()
	handler := &TimeHandler{
		store: visitorStore,
	}
	router.Get("/time", handler.TimeNow())
	router.Get("/visitors", handler.GetVisitors())
}

func (handler *TimeHandler) TimeNow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// generating current time information
		timeCurr := time.Now()
		answer := fmt.Sprint("Current date and time: ", timeCurr.Format("01:02:2006 15:04:05"))
		fmt.Println("Request status:", http.StatusOK)
		fmt.Println(answer)

		// obtaining ip address of the visitor
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = ""
		}
		// storing ip address (if new) and incrementing the count to the map
		handler.store.Add(ip)

		_, err = w.Write([]byte(answer))
		if err != nil {
			log.Error().Msg("Write failed")
		}
	}
}

func (handler *TimeHandler) GetVisitors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		visits := handler.store.GetVisitsAll()
		fmt.Println("Request status:", http.StatusOK)

		// showing the stats by ip address on separate laine
		for ip, count := range visits {
			fmt.Printf("IP address %s has visited the /time page %d times\n", ip, count)
		}
	}
}

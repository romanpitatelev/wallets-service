package currtime

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/romanpitatelev/wallets-service/internal/store"
	"github.com/rs/zerolog/log"
)

type TimeHandler struct {
	store *store.VisitorStore
}

func NewTimeHandler(router *chi.Mux, pool *pgxpool.Pool) {
	visitorStore := store.NewVisitorStore(pool)
	handler := &TimeHandler{
		store: visitorStore,
	}
	router.Get("/time", handler.TimeNow())
	router.Get("/visitors", handler.GetVisitors())
}

func (handler *TimeHandler) TimeNow() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeCurr := time.Now()
		answer := fmt.Sprint("Current date and time: ", timeCurr.Format("01:02:2006 15:04:05"))
		fmt.Println("Request status:", http.StatusOK)
		fmt.Println(answer)

		ipString, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ipString = ""
		}

		mu := sync.Mutex{}
		mu.Lock()
		handler.store.Add(ipString)
		mu.Unlock()

		_, err = w.Write([]byte(answer))
		if err != nil {
			log.Error().Msg("Write failed")
		}
	}
}

func (handler *TimeHandler) GetVisitors() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mu := sync.RWMutex{}
		mu.RLock()
		visits := handler.store.GetVisitsAll()
		mu.RUnlock()

		for ip, count := range visits {
			fmt.Printf("IP address %s has visited the /time page %d times\n", ip, count)
		}
	}
}

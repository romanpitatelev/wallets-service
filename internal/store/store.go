package store

import (
	"github.com/romanpitatelev/wallets-service/internal/ip"
	"gorm.io/gorm"
)

type VisitorStore struct {
	db *gorm.DB
}

func NewVisitorStore(db *gorm.DB) *VisitorStore {
	return &VisitorStore{
		db: db,
	}
}

func (v *VisitorStore) Add(ipAddress string) {
	var ipRecord ip.IP

	v.db.FirstOrCreate(&ipRecord, ip.IP{
		Address: ipAddress,
	})

	ipRecord.Count++
	v.db.Save(&ipRecord)
}

func (v *VisitorStore) GetVisitsAll() map[string]int {
	var ipRecords []ip.IP

	v.db.Find(&ipRecords)

	visits := make(map[string]int)
	for _, record := range ipRecords {
		visits[record.Address] = record.Count
	}

	return visits
}

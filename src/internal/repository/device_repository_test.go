package repository

import (
	"fmt"
	"sync"
	"testing"

	"notion.me/restApi/src/internal/domain"
)

func TestConcurrentAccess(t *testing.T) {
	repo := NewMemoryDeviceRepository()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("dev-%d", i)
			d := domain.Device{Id: id, Status: "online", NodeType: "sensor"}
			if err := repo.Create(d); err != nil {
				t.Error(err)
			}
			if _, ok := repo.GetById(id); !ok {
				t.Error("missing device")
			}
			if !repo.Update(id, d) {
				t.Error("update failed")
			}
			repo.GetAll()
			repo.Filter("online", "sensor")
			if !repo.Delete(id) {
				t.Error("delete failed")
			}
		}(i)
	}
	wg.Wait()
	if len(repo.GetAll()) != 0 {
		t.Fatal("devices remain after deletion")
	}
}

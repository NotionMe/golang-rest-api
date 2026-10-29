package repository

import (
	"fmt"
	"sort"
	"sync"

	"notion.me/restApi/src/internal/domain"
)

type MemoryDeviceRepository struct {
	mu      sync.RWMutex
	storage map[string]domain.Device
}

type DeviceRepository interface {
	GetAll() []domain.Device
	Filter(status, nodeType string) []domain.Device
	GetById(id string) (domain.Device, bool)
	Create(device domain.Device) error
	Update(id string, device domain.Device) bool
	Delete(id string) bool
}

func NewMemoryDeviceRepository() *MemoryDeviceRepository {
	return &MemoryDeviceRepository{
		storage: make(map[string]domain.Device),
	}
}

func (r *MemoryDeviceRepository) GetAll() []domain.Device {
	return r.Filter("", "")
}

func (r *MemoryDeviceRepository) Filter(status, nodeType string) []domain.Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	devices := make([]domain.Device, 0, len(r.storage))

	for _, device := range r.storage {
		if (status == "" || device.Status == status) && (nodeType == "" || device.NodeType == nodeType) {
			devices = append(devices, device)
		}
	}

	sort.Slice(devices, func(i, j int) bool { return devices[i].Id < devices[j].Id })
	return devices
}

func (r *MemoryDeviceRepository) GetById(id string) (domain.Device, bool) {
	r.mu.RLock()

	defer r.mu.RUnlock()

	device, exists := r.storage[id]
	return device, exists
}

func (r *MemoryDeviceRepository) Create(device domain.Device) error {
	r.mu.Lock()

	defer r.mu.Unlock()

	if _, exists := r.storage[device.Id]; exists {
		return fmt.Errorf("Device with id %q already exists", device.Id)
	}

	r.storage[device.Id] = device

	return nil
}

func (r *MemoryDeviceRepository) Update(id string, device domain.Device) bool {
	r.mu.Lock()

	defer r.mu.Unlock()

	if _, exists := r.storage[id]; !exists {
		return false
	}

	device.Id = id
	r.storage[id] = device

	return true
}

func (r *MemoryDeviceRepository) Delete(id string) bool {
	r.mu.Lock()

	defer r.mu.Unlock()

	if _, exists := r.storage[id]; !exists {
		return false
	}

	delete(r.storage, id)

	return true
}

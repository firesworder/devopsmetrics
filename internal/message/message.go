package message

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
)

type Metrics struct {
	ID    string   `json:"id"`              // Имя метрики
	MType string   `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // Значение хеш-функции
}

func (m *Metrics) InitHash(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	h := hmac.New(sha256.New, []byte(key))
	switch m.MType {
	case internal.GaugeTypeName:
		h.Write([]byte(fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)))
	case internal.CounterTypeName:
		h.Write([]byte(fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)))
	default:
		return fmt.Errorf("unhandled type '%s'", m.MType)
	}
	m.Hash = hex.EncodeToString(h.Sum(nil))

	return nil
}

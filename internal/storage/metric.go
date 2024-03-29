package storage

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
)

type gauge float64
type counter int64

// Metric реализует сущность Метрика и методы для работы с ней.
// Сущность имеет название Name, и значение Value(по типу Value определяется и тип метрики - gauge/counter).
type Metric struct {
	Value interface{}
	Name  string
}

// NewMetric конструктор для Metric.
// Создает по аргументам конструктора объект метрики.
// В `rawValue` можно передавать типы: string и int64/float64(для counter/gauge соотв-но).
// typeName поддерживается только "gauge" и "counter", любой другой вызовет ошибку ErrUnhandledValueType.
func NewMetric(name string, typeName string, rawValue interface{}) (*Metric, error) {
	var metricValue interface{}
	switch typeName {
	case internal.CounterTypeName:
		switch castedValue := rawValue.(type) {
		case string:
			valueInt, err := strconv.ParseInt(castedValue, 10, 64)
			if err != nil {
				return nil, err
			}
			metricValue = counter(valueInt)
		case int64:
			metricValue = counter(castedValue)
		default:
			return nil, fmt.Errorf("cannot convert value '%T':'%v' to 'counter' type", rawValue, rawValue)
		}
	case internal.GaugeTypeName:
		switch castedValue := rawValue.(type) {
		case string:
			valueFloat, err := strconv.ParseFloat(castedValue, 64)
			if err != nil {
				return nil, err
			}
			metricValue = gauge(valueFloat)
		case float64:
			metricValue = gauge(castedValue)
		default:
			return nil, fmt.Errorf("cannot convert value '%T':'%v' to 'gauge' type", rawValue, rawValue)
		}
	default:
		return nil, fmt.Errorf("%w '%s'", ErrUnhandledValueType, typeName)
	}
	return &Metric{Name: name, Value: metricValue}, nil
}

// NewMetricFromMessage возвращает объект метрики из message.Metrics.
func NewMetricFromMessage(metrics *message.Metrics) (newMetric *Metric, err error) {
	switch metrics.MType {
	case internal.CounterTypeName:
		if metrics.Delta == nil {
			return nil, fmt.Errorf("param 'delta' cannot be nil for type 'counter'")
		}
		newMetric, err = NewMetric(metrics.ID, metrics.MType, *metrics.Delta)
	case internal.GaugeTypeName:
		if metrics.Value == nil {
			return nil, fmt.Errorf("param 'value' cannot be nil for type 'gauge'")
		}
		newMetric, err = NewMetric(metrics.ID, metrics.MType, *metrics.Value)
	default:
		return nil, fmt.Errorf("%w '%s'", ErrUnhandledValueType, metrics.MType)
	}
	return
}

// Update обновляет значение метрики.
// Для типа "gauge" - перезаписывает значение, для "counter" - прибавляет новое значение к уже существующему.
func (m *Metric) Update(value interface{}) error {
	if reflect.TypeOf(m.Value) != reflect.TypeOf(value) {
		return fmt.Errorf("current(%T) and new(%T) value type mismatch",
			m.Value, value)
	}

	switch value := value.(type) {
	case gauge:
		m.Value = value
	case counter:
		m.Value = m.Value.(counter) + value
	}
	return nil
}

// GetMessageMetric возващает message.Metrics из объекта метрики.
func (m *Metric) GetMessageMetric() (messageMetric message.Metrics) {
	messageMetric.ID = m.Name
	switch value := m.Value.(type) {
	case gauge:
		messageMetric.MType = internal.GaugeTypeName
		mValue := float64(value)
		messageMetric.Value = &mValue
	case counter:
		messageMetric.MType = internal.CounterTypeName
		mDelta := int64(value)
		messageMetric.Delta = &mDelta
	}
	return
}

// GetValueString костыль для прохождения автотестов(инкр. 3b).
func (m *Metric) GetValueString() string {
	switch value := m.Value.(type) {
	case gauge:
		return strings.TrimRight(fmt.Sprintf("%.3f", value), "0")
	case counter:
		return fmt.Sprintf("%d", value)
	}
	return ""
}

// GetMetricParamsString Возвращает параметры метрики в string формате: Name, Value, Type.
func (m *Metric) GetMetricParamsString() (mN string, mV string, mT string) {
	mN = m.Name
	switch value := m.Value.(type) {
	case gauge:
		mT = internal.GaugeTypeName
		mV = fmt.Sprintf("%v", value)
	case counter:
		mT = internal.CounterTypeName
		mV = fmt.Sprintf("%d", value)
	}
	return
}

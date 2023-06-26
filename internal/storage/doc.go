// Package storage это модуль, который обеспечивает хранение и доступ к метрикам, отправленных на этот сервер.
/*
В рамках модуля реализованы:
1) Хранение метрик в памяти(MemStorage) - файлы mem_storage.go + тесты
2) Хранение метрик в SQL DB(SQLStorage) - файл sql_storage.go + тесты
Тип БД - Postgresql.

Оба варианта реализации хранения метрик реализуют интерфейс MetricRepository, который описывает основные методы
используемые вне данного пакета(серверная реализация, прежде всего).

Также реализован тип Metric, позволяющий оперировать сущностью метрики(имя, тип, значение) в коде.
В т.ч. эта сущность используется для описания интерфейса MetricRepository, соотв-но репозитория объектов.
*/
package storage

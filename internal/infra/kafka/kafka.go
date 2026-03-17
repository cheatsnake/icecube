package kafka

import (
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/IBM/sarama"
)

type Producer struct {
	producer sarama.SyncProducer
	topic    string
	logger   *slog.Logger
}

// JobNotification represents a job completion notification sent to Kafka
type JobNotification struct {
	JobID      string    `json:"jobId"`
	Status     string    `json:"status"`
	OriginalID string    `json:"originalId"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// NewProducer creates a new Kafka producer or returns nil if not configured
func NewProducer(brokers, topic string, logger *slog.Logger) *Producer {
	if brokers == "" {
		logger.Debug("Kafka producer not configured, skipping")
		return nil
	}

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 3

	brokerList := parseBrokers(brokers)
	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		logger.Warn("Failed to create Kafka producer", "brokers", brokerList, "error", err)
		return nil
	}

	logger.Info("Kafka producer connected", "brokers", brokerList, "topic", topic)

	return &Producer{
		producer: producer,
		topic:    topic,
		logger:   logger,
	}
}

// SendJobNotification sends a job notification to Kafka asynchronously
func (p *Producer) SendJobNotification(jobID, status, originalID, reason string) {
	if p == nil || p.producer == nil {
		return
	}

	notification := JobNotification{
		JobID:      jobID,
		Status:     status,
		OriginalID: originalID,
		Reason:     reason,
		Timestamp:  time.Now().UTC(),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		p.logger.Warn("Failed to marshal job notification", "jobID", jobID, "error", err)
		return
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(jobID),
		Value: sarama.ByteEncoder(data),
	}

	// Send asynchronously - don't block job processing
	go func() {
		_, _, err := p.producer.SendMessage(msg)
		if err != nil {
			p.logger.Warn("Failed to send job notification", "jobID", jobID, "error", err)
		} else {
			p.logger.Debug("Job notification sent", "jobID", jobID, "status", status)
		}
	}()
}

// Close closes the Kafka producer gracefully
func (p *Producer) Close() {
	if p == nil || p.producer == nil {
		return
	}

	if err := p.producer.Close(); err != nil {
		p.logger.Warn("Error closing Kafka producer", "error", err)
		return
	}

	p.logger.Info("Kafka producer closed")
}

// parseBrokers converts a comma-separated string of brokers to a slice
func parseBrokers(brokers string) []string {
	if brokers == "" {
		return nil
	}

	var result []string
	for _, b := range strings.Split(brokers, ",") {
		trimmed := strings.TrimSpace(b)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

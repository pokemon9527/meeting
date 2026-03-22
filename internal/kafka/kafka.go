package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	TopicRecordingRaw        = "recording-raw"
	TopicRecordingTranscoded = "recording-transcoded"
)

type RawSegmentMessage struct {
	RecordingID     string    `json:"recording_id"`
	MeetingID       string    `json:"meeting_id"`
	SegmentID       string    `json:"segment_id"`
	ParticipantID   string    `json:"participant_id"`
	ParticipantName string    `json:"participant_name"`
	SegmentPath     string    `json:"segment_path"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	FileSize        int64     `json:"file_size"`
	SequenceNumber  int       `json:"sequence_number"`
}

type TranscodedMessage struct {
	RecordingID     string `json:"recording_id"`
	SegmentID       string `json:"segment_id"`
	Quality         string `json:"quality"`
	PlaylistPath    string `json:"playlist_path"`
	TotalSegments   int    `json:"total_segments"`
	TotalSize       int64  `json:"total_size"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		RequiredAcks: kafka.RequireOne,
	}
	return &Producer{writer: writer}
}

func (p *Producer) PublishRawSegment(ctx context.Context, msg RawSegmentMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.SegmentID),
		Value: data,
	})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published raw segment message: %s", msg.SegmentID)
	return nil
}

func (p *Producer) PublishTranscoded(ctx context.Context, msg TranscodedMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.SegmentID),
		Value: data,
	})
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Published transcoded message: %s", msg.SegmentID)
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	return &Consumer{reader: reader}
}

func (c *Consumer) Consume(ctx context.Context, handler func([]byte) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			err = handler(msg.Value)
			if err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func EnsureTopics(brokers []string, topics []string) error {
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("failed to get controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("failed to connect to controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{}
	for _, topic := range topics {
		topicConfigs = append(topicConfigs, kafka.TopicConfig{
			Topic:             topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		})
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		log.Printf("Topic creation warning (may already exist): %v", err)
	}

	return nil
}

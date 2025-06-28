package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/aichat-backend/internal/models"
)

// StreamInfo holds stream data and ownership information
type StreamInfo struct {
	UserID     string
	StreamChan <-chan *models.StreamMessage
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// StreamManager handles stream lifecycle and storage
type StreamManager struct {
	activeStreams map[string]*StreamInfo
	streamsMu     sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	l             *logrus.Entry
}

// NewStreamManager creates a new stream manager
func NewStreamManager() *StreamManager {
	sm := &StreamManager{
		activeStreams: make(map[string]*StreamInfo),
		stopCleanup:   make(chan struct{}),
		l:             logrus.WithField("component", "stream-manager"),
	}

	// Start periodic cleanup in the background
	sm.startPeriodicCleanup()

	return sm
}

// StoreStream stores a stream channel with the given stream ID and user ownership
// Streams automatically expire after 10 minutes
func (sm *StreamManager) StoreStream(streamID, userID string, streamChan <-chan *models.StreamMessage) {
	sm.streamsMu.Lock()
	defer sm.streamsMu.Unlock()
	now := time.Now()
	expiresAt := now.Add(10 * time.Minute)
	sm.activeStreams[streamID] = &StreamInfo{
		UserID:     userID,
		StreamChan: streamChan,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
	}
	sm.l.WithFields(logrus.Fields{
		"stream_id":  streamID,
		"user_id":    userID,
		"expires_at": expiresAt.Format("15:04:05"),
	}).Debug("Stored stream channel")
}

// GetStreamForUser retrieves a stream channel by stream ID and validates user ownership
func (sm *StreamManager) GetStreamForUser(streamID, userID string) (<-chan *models.StreamMessage, error) {
	sm.streamsMu.Lock() // Use write lock to allow cleanup of expired streams
	defer sm.streamsMu.Unlock()

	streamInfo, exists := sm.activeStreams[streamID]
	if !exists {
		sm.l.WithField("stream_id", streamID).Debug("Stream not found")
		return nil, fmt.Errorf("stream not found: %s", streamID)
	}

	// Check if stream has expired
	if time.Now().After(streamInfo.ExpiresAt) {
		sm.l.WithFields(logrus.Fields{
			"stream_id":  streamID,
			"expired_at": streamInfo.ExpiresAt.Format("15:04:05"),
		}).Debug("Stream expired, removing")
		delete(sm.activeStreams, streamID)
		return nil, fmt.Errorf("stream expired: %s", streamID)
	}

	if streamInfo.UserID != userID {
		sm.l.WithFields(logrus.Fields{
			"stream_id":    streamID,
			"user_id":      userID,
			"stream_owner": streamInfo.UserID,
		}).Warn("Access denied: user cannot access stream owned by different user")
		return nil, fmt.Errorf("access denied: stream belongs to different user")
	}

	sm.l.WithFields(logrus.Fields{
		"stream_id": streamID,
		"user_id":   userID,
	}).Debug("Retrieved stream channel")
	return streamInfo.StreamChan, nil
}

// CloseStream removes a stream from active streams
func (sm *StreamManager) CloseStream(streamID string) {
	sm.streamsMu.Lock()
	defer sm.streamsMu.Unlock()
	delete(sm.activeStreams, streamID)
	sm.l.WithField("stream_id", streamID).Debug("Closed and removed stream")
}

// InitiateStream creates a new stream and returns its ID
func (sm *StreamManager) InitiateStream(ctx context.Context, userID string, streamChan <-chan *models.StreamMessage) string {
	streamID := fmt.Sprintf("stream_%s_%d", userID, time.Now().UnixNano())
	sm.StoreStream(streamID, userID, streamChan)
	return streamID
}

// ConnectToStream connects to an existing stream by ID for a specific user
func (sm *StreamManager) ConnectToStream(ctx context.Context, userID, streamID string) (<-chan *models.StreamMessage, error) {
	streamChan, err := sm.GetStreamForUser(streamID, userID)
	if err != nil {
		return nil, err
	}

	sm.l.WithFields(logrus.Fields{
		"user_id":   userID,
		"stream_id": streamID,
	}).Debug("User connected to stream")
	return streamChan, nil
}

// CleanupExpiredStreams removes expired streams from active streams
func (sm *StreamManager) CleanupExpiredStreams() {
	sm.streamsMu.Lock()
	defer sm.streamsMu.Unlock()

	now := time.Now()
	expiredCount := 0

	for streamID, streamInfo := range sm.activeStreams {
		if now.After(streamInfo.ExpiresAt) {
			delete(sm.activeStreams, streamID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		sm.l.WithField("expired_streams_count", expiredCount).Info("Cleaned up expired streams")
	}
}

// startPeriodicCleanup starts a background goroutine to periodically clean up expired streams
func (sm *StreamManager) startPeriodicCleanup() {
	sm.cleanupTicker = time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes

	go func() {
		sm.l.Info("Starting periodic stream cleanup (every 5 minutes)")

		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.CleanupExpiredStreams()
			case <-sm.stopCleanup:
				sm.cleanupTicker.Stop()
				sm.l.Info("Periodic stream cleanup stopped")
				return
			}
		}
	}()
}

// Close cleans up all active streams and stops the cleanup ticker
func (sm *StreamManager) Close() error {
	// Stop the periodic cleanup
	close(sm.stopCleanup)

	sm.streamsMu.Lock()
	defer sm.streamsMu.Unlock()

	streamCount := len(sm.activeStreams)
	sm.activeStreams = make(map[string]*StreamInfo)

	sm.l.WithField("closed_streams_count", streamCount).Info("StreamManager closed all active streams")
	return nil
}

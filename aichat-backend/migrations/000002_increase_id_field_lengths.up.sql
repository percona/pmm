-- Increase ID field lengths to accommodate longer session IDs

-- Increase session ID length
ALTER TABLE chat_sessions ALTER COLUMN id TYPE VARCHAR(64);

-- Increase message ID and session_id reference length
ALTER TABLE chat_messages ALTER COLUMN id TYPE VARCHAR(64);
ALTER TABLE chat_messages ALTER COLUMN session_id TYPE VARCHAR(64);

-- Increase attachment ID and message_id reference length  
ALTER TABLE chat_attachments ALTER COLUMN id TYPE VARCHAR(64);
ALTER TABLE chat_attachments ALTER COLUMN message_id TYPE VARCHAR(64); 
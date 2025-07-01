-- Revert ID field lengths back to original size
-- WARNING: This may fail if there are existing records with longer IDs

-- Revert attachment ID and message_id reference length  
ALTER TABLE chat_attachments ALTER COLUMN message_id TYPE VARCHAR(36);
ALTER TABLE chat_attachments ALTER COLUMN id TYPE VARCHAR(36);

-- Revert message ID and session_id reference length
ALTER TABLE chat_messages ALTER COLUMN session_id TYPE VARCHAR(36);
ALTER TABLE chat_messages ALTER COLUMN id TYPE VARCHAR(36);

-- Revert session ID length
ALTER TABLE chat_sessions ALTER COLUMN id TYPE VARCHAR(36); 
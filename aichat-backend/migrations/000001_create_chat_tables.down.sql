-- Drop chat tables for AI chat backend
 
-- Drop tables in reverse order due to foreign key constraints
DROP TABLE IF EXISTS chat_attachments;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions; 
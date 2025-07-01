/**
 * Utility functions for session ID generation
 */

/**
 * Generates a short, unique session ID for analysis sessions
 * Format: prefix_XXXXXXXX (8-digit suffix from timestamp)
 * Example: analysis_12345678, overview_87654321
 */
export function generateAnalysisSessionId(prefix: string): string {
  const timestamp = Date.now().toString();
  const shortSuffix = timestamp.slice(-8); // Last 8 digits
  return `${prefix}_${shortSuffix}`;
}

/**
 * Generates a UUID-like session ID for regular chat sessions
 * Uses crypto.randomUUID() if available, falls back to timestamp-based generation
 */
export function generateChatSessionId(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  
  // Fallback for environments without crypto.randomUUID()
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substring(2, 8);
  return `session_${timestamp}_${random}`;
}

/**
 * Validates if a session ID is properly formatted
 */
export function isValidSessionId(sessionId: string): boolean {
  if (!sessionId || typeof sessionId !== 'string') {
    return false;
  }
  
  // Should be between 3 and 64 characters (matching our DB constraint)
  if (sessionId.length < 3 || sessionId.length > 64) {
    return false;
  }
  
  // Should contain only alphanumeric characters, hyphens, and underscores
  const validPattern = /^[a-zA-Z0-9_-]+$/;
  return validPattern.test(sessionId);
} 
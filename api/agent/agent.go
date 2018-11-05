package agent

// Workaround for https://github.com/golang/protobuf/issues/261.
// Useful for helper functions.
type (
	ServerMessagePayload = isServerMessage_Payload
	AgentMessagePayload  = isAgentMessage_Payload
)

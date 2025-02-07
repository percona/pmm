import { useState } from 'react';
import { Message, useMessenger } from './messages.provider';

export const useMessageWithResult = () => {
  const [id, setId] = useState<string>();
  const { messages, sendMessage } = useMessenger();
  const result = messages.find((msg) => msg.data?.id === id);

  const sendMessageWithResult = (msg: Message) => {
    const id = self.crypto.randomUUID();
    setId(id);
    sendMessage({
      ...msg,
      data: {
        ...msg.data,
        id,
      },
    });
  };

  return { result, sendMessage: sendMessageWithResult };
};

export const useMessages = (filter: string) => {
  const { messages } = useMessenger();
  return messages.filter((m) => !filter || m.type === filter);
};
